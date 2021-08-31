// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package realue

import (
	intfc "gnbsim/interfacecommon"
	"gnbsim/logger"
	"gnbsim/realue/context"
	"gnbsim/util/test"

	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/omec-project/nas"
)

func Init(ue *context.RealUe) {
	if ue == nil {
		logger.RealUeLog.Errorln("RealUe is nil")
		return
	}
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC,
		"")

	for msg := range ue.ReadChan {
		err := HandleEvent(ue, msg)
		if err != nil {
			ue.Log.Errorln("Failed to handle received event")
		}
	}
}

func HandleEvent(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	ue.Log.Infoln("Handling Event:", msg.Event)

	switch msg.Event {
	case intfc.UE_REG_REQUEST:
		err = HandleRegReqEvent(ue, msg)
	case intfc.UE_AUTH_RESPONSE:
		err = HandleAuthResponseEvent(ue, msg)
	case intfc.UE_SEC_MOD_COMPLETE:
		err = HandleSecModCompleteEvent(ue, msg)
	case intfc.UE_REG_COMPLETE:
		err = HandleRegCompleteEvent(ue, msg)
	case intfc.AMF_DOWNLINK_NAS_TRANSPORT:
		err = HandleDownlinkNasTransportEvent(ue, msg)
	default:
		ue.Log.Infoln("Event", msg.Event, "is not supported")
	}

	if err != nil {
		ue.Log.Errorln("Failed to process event:", msg.Event, "Error:", err)
	}

	return err
}

func SendToSimUe(ue *context.RealUe, event intfc.EventType, naspdu []byte,
	nasmsg *nas.Message) {
	ue.Log.Infoln("Sending event", event, "to SimUe")
	msg := &intfc.UuMessage{}
	msg.Event = event
	msg.NasPdu = naspdu
	msg.Extras.NasMsg = nasmsg
	ue.WriteSimUeChan <- msg
}
