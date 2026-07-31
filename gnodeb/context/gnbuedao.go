// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"

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

// GetGnbUpUe returns the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) GetGnbUpUe(teid uint32, downlink bool) *GnbUpUe {
	dao.Log.Debugf("fetching GnbUpUe for TEID: %d downlink: %v", teid, downlink)
	var val interface{}
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
	dao.Log.Infoln("enqueueing pending handover target GnbCpUe")
	select {
	case dao.pendingHandoverTargets <- gnbue:
	default:
		dao.Log.Errorln("pending handover target queue full; dropping pre-registered target GnbCpUe")
	}
}

// DequeueHandoverTarget retrieves and removes the next pre-registered handover
// target GnbCpUe. Returns nil immediately if the queue is empty.
func (dao *GnbUeDao) DequeueHandoverTarget() *GnbCpUe {
	select {
	case gnbue := <-dao.pendingHandoverTargets:
		dao.Log.Infoln("dequeued pending handover target GnbCpUe")
		return gnbue
	default:
		dao.Log.Warnln("no pending handover target GnbCpUe in queue")
		return nil
	}
}

// GetGnbCpUeByUeChan finds a GnbCpUe whose WriteUeChan matches the given channel.
// Used during N2 handover preparation to retrieve the source GnbCpUe's AMF UE NGAP ID.
func (dao *GnbUeDao) GetGnbCpUeByUeChan(ch chan common.InterfaceMessage) *GnbCpUe {
	var found *GnbCpUe
	dao.ngapIdGnbCpUeMap.Range(func(k, v interface{}) bool {
		gnbue := v.(*GnbCpUe)
		if gnbue.WriteUeChan == ch {
			found = gnbue
			return false
		}
		return true
	})
	return found
}
