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

func HandleEvent(ue *context.RealUe, msg common.InterfaceMessage) (err error) {
	if msg == nil {
		return fmt.Errorf("empty message received")
	}

	event := msg.GetEventType()
	ue.Log.Traceln("Handling Event:", event)

	/* TODO: Should check interface type to avoid overlapping events
	 * add support for N1 interface, internal realue-sim ue interface
	 */
	switch event {
	case common.REG_REQUEST_EVENT:
		err = HandleRegRequestEvent(ue, msg)
	case common.AUTH_RESPONSE_EVENT:
		err = HandleAuthResponseEvent(ue, msg)
	case common.SEC_MOD_COMPLETE_EVENT:
		err = HandleSecModCompleteEvent(ue, msg)
	case common.REG_COMPLETE_EVENT:
		err = HandleRegCompleteEvent(ue, msg)
	case common.DL_INFO_TRANSFER_EVENT:
		err = HandleDlInfoTransferEvent(ue, msg)
	case common.PDU_SESS_EST_REQUEST_EVENT:
		err = HandlePduSessEstRequestEvent(ue, msg)
	case common.PDU_SESS_EST_ACCEPT_EVENT:
		err = HandlePduSessEstAcceptEvent(ue, msg)
	case common.DATA_BEARER_SETUP_REQUEST_EVENT:
		err = HandleDataBearerSetupRequestEvent(ue, msg)
	case common.DATA_PKT_GEN_REQUEST_EVENT:
		err = HandleDataPktGenRequestEvent(ue, msg)
	case common.DATA_PKT_GEN_SUCCESS_EVENT:
		err = HandleDataPktGenSuccessEvent(ue, msg)
	default:
		ue.Log.Infoln("Event", event, "is not supported")
	}

	if err != nil {
		ue.Log.Errorln("Failed to process event:", event, "Error:", err)
	}

	return err
}

func formUuMessage(event common.EventType, nasPdu []byte) *common.UuMessage {
	msg := &common.UuMessage{}
	msg.Event = event
	msg.NasPdus = append(msg.NasPdus, nasPdu)
	return msg
}

func SendToSimUe(ue *context.RealUe,
	msg common.InterfaceMessage) {

	ue.Log.Infoln("Sending event", msg.GetEventType(), "to SimUe")
	ue.WriteSimUeChan <- msg
}
