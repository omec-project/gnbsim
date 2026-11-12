// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"bytes"
	"encoding/binary"
	"sync"
	"sync/atomic"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"go.uber.org/zap"
)

// TODO: Need to separate out the DAOs

// GnbUeDao acts as a Data Access Object that stores and provides access to all
// the GNodeB instances
type GnbUeDao struct {
	Log                    *zap.SugaredLogger
	pendingHandoverTargets chan *GnbCpUe
	ngapIdGnbCpUeMap       sync.Map
	dlTeidGnbUpUeMap       sync.Map
	pendingMu              sync.Mutex
	handoverNonce          atomic.Uint64
}

func NewGnbUeDao() *GnbUeDao {
	dao := &GnbUeDao{}
	dao.Log = logger.GNodeBLog.With("subcategory", "GnbUeDao")
	// Buffer size of 8 is more than enough for simulation workloads.
	dao.pendingHandoverTargets = make(chan *GnbCpUe, 8)
	return dao
}

// GetGnbCpUe returns the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbUeDao) GetGnbCpUe(gnbUeNgapId int64) *GnbCpUe {
	dao.Log.Infoln("fetching GnbCpUe for RANUENGAPID:", gnbUeNgapId)
	val, ok := dao.ngapIdGnbCpUeMap.Load(gnbUeNgapId)
	if ok {
		return val.(*GnbCpUe)
	} else {
		dao.Log.Warnln("key not present:", gnbUeNgapId)
		return nil
	}
}

// AddGnbCpUe adds the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbUeDao) AddGnbCpUe(gnbUeNgapId int64, gnbue *GnbCpUe) {
	dao.Log.Infoln("adding new GnbCpUe for RANUENGAPID:", gnbUeNgapId)
	dao.ngapIdGnbCpUeMap.Store(gnbUeNgapId, gnbue)
}

// RemoveGnbCpUe removes the GnbCpUe instance corresponding to provided NGAP ID, so that messages
// from the AMF for this ID are no longer routed to it.
func (dao *GnbUeDao) RemoveGnbCpUe(gnbUeNgapId int64) {
	dao.Log.Infoln("removing GnbCpUe for RANUENGAPID:", gnbUeNgapId)
	dao.ngapIdGnbCpUeMap.Delete(gnbUeNgapId)
	dao.pruneCancelledHandoverTargets()
}

// pruneCancelledHandoverTargets drops the pending handover targets that are no longer registered,
// keeping the order of the others. Without it a target whose request never comes would keep its
// place in the small queue until some other request happened to be processed, and enough of them
// would make the next registration be dropped.
func (dao *GnbUeDao) pruneCancelledHandoverTargets() {
	dao.pendingMu.Lock()
	defer dao.pendingMu.Unlock()
	var keep []*GnbCpUe
	for {
		select {
		case gnbue := <-dao.pendingHandoverTargets:
			if registered, ok := dao.ngapIdGnbCpUeMap.Load(gnbue.GnbUeNgapId); ok && registered == gnbue {
				keep = append(keep, gnbue)
			}
			continue
		default:
		}
		break
	}
	for _, gnbue := range keep {
		dao.pendingHandoverTargets <- gnbue // at most what was taken out, so there is room
	}
}

// GetGnbUpUe returns the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) GetGnbUpUe(teid uint32, downlink bool) *GnbUpUe {
	dao.Log.Debugf("fetching GnbUpUe for TEID: %d downlink: %v", teid, downlink)
	var val any
	var ok bool
	if downlink {
		val, ok = dao.dlTeidGnbUpUeMap.Load(teid)
	}

	if ok {
		return val.(*GnbUpUe)
	} else {
		dao.Log.Warnln("key not present:", teid, "Downlink TEID :", downlink)
		return nil
	}
}

// AddGnbUpUe adds the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) AddGnbUpUe(teid uint32, downlink bool, gnbue *GnbUpUe) {
	dao.Log.Infoln("adding new GnbUpUe for TEID:", teid, "Downlink:", downlink)
	if downlink {
		dao.dlTeidGnbUpUeMap.Store(teid, gnbue)
	}
}

// RemoveGnbUpUe removes the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) RemoveGnbUpUe(teid uint32, downlink bool) {
	dao.Log.Infoln("removing GnbUpUe for TEID:", teid, "Downlink:", downlink)
	if downlink {
		dao.dlTeidGnbUpUeMap.Delete(teid)
	}
}

// EnqueueHandoverTarget stores a pre-registered target GnbCpUe in the pending
// handover queue so gnbamfworker can retrieve it when HandoverRequest arrives.
func (dao *GnbUeDao) EnqueueHandoverTarget(gnbue *GnbCpUe) {
	dao.pendingMu.Lock()
	defer dao.pendingMu.Unlock()
	dao.Log.Infoln("enqueueing pending handover target GnbCpUe")
	select {
	case dao.pendingHandoverTargets <- gnbue:
	default:
		dao.Log.Errorln("pending handover target queue full; dropping pre-registered target GnbCpUe")
	}
}

// handoverTokenMagic starts the Source to Target Transparent Container that this gNB puts in a
// HandoverRequired. The container is opaque to the AMF, which hands it to the target unchanged,
// so the target gNB can read from it which pre-registered target the request is for.
var handoverTokenMagic = []byte{0x47, 0x4e, 0x42, 0x01}

// NextHandoverNonce returns a value that no other registration of a handover target has had.
func (dao *GnbUeDao) NextHandoverNonce() uint64 {
	return dao.handoverNonce.Add(1)
}

// HandoverTargetToken returns the container that names one registration of a pre-registered
// target: the RAN UE NGAP ID on the target gNB, which is reused once a UE quits, and the nonce of
// that registration, which is not.
func HandoverTargetToken(targetGnbUeNgapId int64, nonce uint64) []byte {
	token := append([]byte(nil), handoverTokenMagic...)
	token = binary.BigEndian.AppendUint64(token, uint64(targetGnbUeNgapId))
	return binary.BigEndian.AppendUint64(token, nonce)
}

// ParseHandoverTargetToken returns the RAN UE NGAP ID and the nonce that a container made by
// HandoverTargetToken names, and false if the container is not one.
func ParseHandoverTargetToken(container []byte) (int64, uint64, bool) {
	if len(container) != len(handoverTokenMagic)+16 || !bytes.HasPrefix(container, handoverTokenMagic) {
		return 0, 0, false
	}
	rest := container[len(handoverTokenMagic):]
	return int64(binary.BigEndian.Uint64(rest)), binary.BigEndian.Uint64(rest[8:]), true
}

// TakePendingHandoverTarget removes and returns the pre-registered target that a HandoverRequest
// names in its container, by RAN UE NGAP ID and registration nonce. It returns nil if there is
// none, for instance because its UE quit, in which case the request has no one to go to; a later
// registration that was given the same ID is not that target. Entries of other targets are left
// queued, and those that have been unregistered are dropped.
func (dao *GnbUeDao) TakePendingHandoverTarget(gnbUeNgapId int64, nonce uint64) *GnbCpUe {
	dao.pendingMu.Lock()
	defer dao.pendingMu.Unlock()
	var found *GnbCpUe
	var keep []*GnbCpUe
	for {
		select {
		case gnbue := <-dao.pendingHandoverTargets:
			if registered, ok := dao.ngapIdGnbCpUeMap.Load(gnbue.GnbUeNgapId); !ok || registered != gnbue {
				continue // cancelled, nothing will ask for it
			}
			if gnbue.GnbUeNgapId == gnbUeNgapId && gnbue.HandoverNonce == nonce && found == nil {
				found = gnbue
				continue
			}
			keep = append(keep, gnbue)
			continue
		default:
		}
		break
	}
	for _, gnbue := range keep {
		dao.pendingHandoverTargets <- gnbue // at most what was taken out, so there is room
	}
	return found
}

// DequeueHandoverTarget retrieves and removes the next pre-registered handover
// target GnbCpUe. Returns nil immediately if the queue is empty. It is for a HandoverRequest
// whose container does not name its target (see TakePendingHandoverTarget), which can only be
// matched to targets in order. A target that has been unregistered is skipped.
func (dao *GnbUeDao) DequeueHandoverTarget() *GnbCpUe {
	dao.pendingMu.Lock()
	defer dao.pendingMu.Unlock()
	for {
		select {
		case gnbue := <-dao.pendingHandoverTargets:
			if registered, ok := dao.ngapIdGnbCpUeMap.Load(gnbue.GnbUeNgapId); !ok || registered != gnbue {
				dao.Log.Warnln("skipping cancelled pending handover target GnbCpUe")
				continue
			}
			dao.Log.Infoln("dequeued pending handover target GnbCpUe")
			return gnbue
		default:
			dao.Log.Warnln("no pending handover target GnbCpUe in queue")
			return nil
		}
	}
}

// GetGnbCpUeByUeChan finds a GnbCpUe whose WriteUeChan matches the given channel.
// Used during N2 handover preparation to retrieve the source GnbCpUe's AMF UE NGAP ID.
func (dao *GnbUeDao) GetGnbCpUeByUeChan(ch chan common.InterfaceMessage) *GnbCpUe {
	var found *GnbCpUe
	dao.ngapIdGnbCpUeMap.Range(func(k, v any) bool {
		gnbue := v.(*GnbCpUe)
		if gnbue.WriteUeChan == ch {
			found = gnbue
			return false
		}
		return true
	})
	return found
}
