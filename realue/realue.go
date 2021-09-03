// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package realue

import (
	"fmt"
	"gnbsim/common"
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
			ue.Log.Errorln("Failed to handle received event", err)
		}
	}
}

func HandleEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	if msg == nil {
		return fmt.Errorf("empty message received")
	}

	ue.Log.Traceln("Handling Event:", msg.Event)

	switch msg.Event {
	case common.REG_REQUEST_EVENT:
		err = HandleRegReqEvent(ue, msg)
	case common.AUTH_RESPONSE_EVENT:
		err = HandleAuthResponseEvent(ue, msg)
	case common.SEC_MOD_COMPLETE_EVENT:
		err = HandleSecModCompleteEvent(ue, msg)
	case common.REG_COMPLETE_EVENT:
		err = HandleRegCompleteEvent(ue, msg)
	case common.DL_INFO_TRANSFER_EVENT:
		err = HandleDlInfoTransferEvent(ue, msg)
	default:
		ue.Log.Infoln("Event", msg.Event, "is not supported")
	}

	if err != nil {
		ue.Log.Errorln("Failed to process event:", msg.Event, "Error:", err)
	}

	return err
}

func SendToSimUe(ue *context.RealUe, event common.EventType, naspdu []byte,
	nasmsg *nas.Message) {
	ue.Log.Infoln("Sending event", event, "to SimUe")
	msg := &common.UuMessage{}
	msg.Event = event
	msg.Interface = common.UU_INTERFACE
	msg.NasPdu = naspdu
	msg.Extras.NasMsg = nasmsg
	ue.WriteSimUeChan <- msg
}
