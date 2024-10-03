// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/ngap/ngapType"
	"github.com/omec-project/openapi/models"
	"go.uber.org/zap"
)

type GnbUpUe struct {
	Upf         *GnbUpf
	Gnb         *GNodeB
	Log         *zap.SugaredLogger
	PduSessType models.PduSessionType
	QosFlows    map[int64]*ngapType.QosFlowSetupRequestItem

	// GnbUpUe writes downlink packets to UE on this channel
	WriteUeChan chan common.InterfaceMessage

	// GnbUpUe reads up link data packets from UE on this channel
	ReadUlChan chan common.InterfaceMessage

	// GnbUpUe reads down link data packets from UPF Worker on this channel
	ReadDlChan chan common.InterfaceMessage

	// GnbUpUe reads commands from GnbCpUe on this channel
	ReadCmdChan chan common.InterfaceMessage

	Snssai           models.Snssai
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
	ue.QosFlows[qfi] = qosFlow
}
