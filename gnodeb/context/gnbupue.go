// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/ngap/v2/ngapType"
	"github.com/omec-project/openapi/v2/models"
	"go.uber.org/zap"
)

type GnbUpUe struct {
	Snssai models.Snssai
	// GnbUpUe reads up link data packets from UE on this channel
	ReadUlChan chan common.InterfaceMessage
	Log        *zap.SugaredLogger
	// QosFlows is reached through AddQosFlow, RemoveQosFlow, GetQosFlow and AnyQfi, which hold
	// qosFlowsMu.
	// The control plane records an admitted flow on its own goroutine, and from the moment a
	// PDU session resource modify request can arrive that is concurrent with this session's
	// user plane worker reading the map for every uplink packet.
	QosFlows map[int64]*ngapType.QosFlowSetupRequestItem
	// GnbUpUe writes downlink packets to UE on this channel
	WriteUeChan chan common.InterfaceMessage
	Upf         *GnbUpf
	// GnbUpUe reads down link data packets from UPF Worker on this channel
	ReadDlChan chan common.InterfaceMessage
	// GnbUpUe reads commands from GnbCpUe on this channel
	ReadCmdChan chan common.InterfaceMessage
	Gnb         *GNodeB
	PduSessType models.PduSessionType
	// qosFlowsMu guards QosFlows. It sits down here, away from the map, because fieldalignment
	// wants the pointer-bearing fields together at the top.
	qosFlowsMu       sync.RWMutex
	PduSessId        int64
	DlTeid           uint32
	UlTeid           uint32
	LastDataPktRecvd bool
}

func NewGnbUpUe(dlTeid, ulTeid uint32, gnb *GNodeB) *GnbUpUe {
	gnbue := GnbUpUe{}
	gnbue.DlTeid = dlTeid
	gnbue.UlTeid = ulTeid
	gnbue.Gnb = gnb
	gnbue.QosFlows = make(map[int64]*ngapType.QosFlowSetupRequestItem)
	gnbue.ReadUlChan = make(chan common.InterfaceMessage, 10)
	gnbue.ReadDlChan = make(chan common.InterfaceMessage, 10)
	gnbue.ReadCmdChan = make(chan common.InterfaceMessage, 5)
	gnbue.Log = logger.GNodeBLog.With("subcategory", "GnbUpUe", logger.FieldDlTeid, dlTeid)
	gnbue.Log.Debugln("context created")
	return &gnbue
}

func (ue *GnbUpUe) GetQosFlow(qfi int64) *ngapType.QosFlowSetupRequestItem {
	ue.Log.Infoln("fetching QosFlowItem corresponding to QFI:", qfi)
	ue.qosFlowsMu.RLock()
	defer ue.qosFlowsMu.RUnlock()
	val, ok := ue.QosFlows[qfi]
	if ok {
		return val
	} else {
		ue.Log.Errorln("no QOS Flow found corresponding to QFI:", qfi)
		return nil
	}
}

func (ue *GnbUpUe) AddQosFlow(qfi int64, qosFlow *ngapType.QosFlowSetupRequestItem) {
	ue.Log.Infoln("adding new QosFlowItem corresponding to QFI:", qfi)
	ue.qosFlowsMu.Lock()
	defer ue.qosFlowsMu.Unlock()
	ue.QosFlows[qfi] = qosFlow
}

// RemoveQosFlow drops a QoS flow the core has released from this session's view.
//
// Keeping it would leave the gNB serving a flow the network has withdrawn, and AnyQfi would go on
// offering its QFI to the uplink path after the user plane has stopped expecting it.
func (ue *GnbUpUe) RemoveQosFlow(qfi int64) {
	ue.Log.Infoln("removing QosFlowItem corresponding to QFI:", qfi)
	ue.qosFlowsMu.Lock()
	defer ue.qosFlowsMu.Unlock()
	delete(ue.QosFlows, qfi)
}

// AnyQfi returns one of the QFIs recorded for this session, and whether there was one.
//
// The uplink path needs a QFI to stamp on the GTP-U header and any of the session's flows will
// do. It is an accessor rather than a range at the call site because the control plane can add a
// flow while that range runs: the network starts a modification whenever it decides to, which
// includes the middle of a data transfer.
//
// The second return value is what distinguishes a session whose only flow is QFI 0 from one that
// has no flow at all -- which a modification releasing the last flow now produces, and which the
// range this replaced reported as QFI 0 either way.
func (ue *GnbUpUe) AnyQfi() (int64, bool) {
	ue.qosFlowsMu.RLock()
	defer ue.qosFlowsMu.RUnlock()
	for qfi := range ue.QosFlows {
		return qfi, true
	}
	return 0, false
}
