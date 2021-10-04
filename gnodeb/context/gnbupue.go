// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"gnbsim/common"
	"gnbsim/logger"

	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

type GnbUpUe struct {
	PduSessId   int64
	DlTeid      uint32
	UlTeid      uint32
	Snssai      models.Snssai
	Upf         *GnbUpf
	Gnb         *GNodeB
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

	/* logger */
	Log *logrus.Entry
}

func NewGnbUpUe(dlTeid, ulTeid uint32, gnb *GNodeB) *GnbUpUe {
	gnbue := GnbUpUe{}
	gnbue.DlTeid = dlTeid
	gnbue.UlTeid = ulTeid
	gnbue.Gnb = gnb
	gnbue.QosFlows = make(map[int64]*ngapType.QosFlowSetupRequestItem)
	gnbue.ReadUlChan = make(chan common.InterfaceMessage, 10)
	gnbue.ReadDlChan = make(chan common.InterfaceMessage, 10)
	gnbue.ReadCmdChan = make(chan common.InterfaceMessage)
	gnbue.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbUpUe",
		logger.FieldDlTeid: dlTeid})
	gnbue.Log.Traceln("Context Created")
	return &gnbue
}

func (ue *GnbUpUe) GetQosFlow(qfi int64) *ngapType.QosFlowSetupRequestItem {
	ue.Log.Infoln("Fetching QosFlowItem corresponding to QFI:", qfi)
	val, ok := ue.QosFlows[qfi]
	if ok {
		return val
	} else {
		ue.Log.Errorln("No QOS Flow found corresponding to QFI:", qfi)
		return nil
	}
}

func (ue *GnbUpUe) AddQosFlow(qfi int64, qosFlow *ngapType.QosFlowSetupRequestItem) {
	ue.Log.Infoln("Adding new QosFlowItem corresponding to QFI:", qfi)
	ue.QosFlows[qfi] = qosFlow
}
