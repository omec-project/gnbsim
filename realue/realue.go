// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package realue

import (
	"gnbsim/common"
	"gnbsim/realue/context"
	"gnbsim/util/test"
	"sync"

	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
)

func Init(ue *context.RealUe, wg *sync.WaitGroup) {

	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC,
		"")

	HandleEvents(ue)
	wg.Done()
}

func HandleEvents(ue *context.RealUe) (err error) {

	for msg := range ue.ReadChan {
		event := msg.GetEventType()
		evtStr := common.GetEvtString(event)
		ue.Log.Infoln("Handling:", evtStr)

		switch event {
		case common.REG_REQUEST_EVENT:
			err = HandleRegRequestEvent(ue, msg)
		case common.AUTH_RESPONSE_EVENT:
			err = HandleAuthResponseEvent(ue, msg)
		case common.SEC_MOD_COMPLETE_EVENT:
			err = HandleSecModCompleteEvent(ue, msg)
		case common.REG_COMPLETE_EVENT:
			err = HandleRegCompleteEvent(ue, msg)
		case common.DEREG_REQUEST_UE_ORIG_EVENT:
			err = HandleDeregRequestEvent(ue, msg)
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
		case common.SERVICE_REQUEST_EVENT:
			err = HandleServiceRequestEvent(ue, msg)
		case common.CONNECTION_RELEASE_REQUEST_EVENT:
			err = HandleConnectionReleaseRequestEvent(ue, msg)
		case common.ERROR_EVENT:
			HandleErrorEvent(ue, msg)
		case common.QUIT_EVENT:
			HandleQuitEvent(ue, msg)
			return nil
		default:
			ue.Log.Warnln("Event", evtStr, "is not supported")
		}

		if err != nil {
			ue.Log.Errorln("real ue failed:", evtStr, ":", err)
			msg := &common.UeMessage{}
			msg.Error = err
			msg.Event = common.ERROR_EVENT
			HandleErrorEvent(ue, msg)
		}
	}

	return nil
}

func formUuMessage(event common.EventType, nasPdu []byte) *common.UuMessage {
	msg := &common.UuMessage{}
	msg.Event = event
	msg.NasPdus = append(msg.NasPdus, nasPdu)
	return msg
}

func SendToSimUe(ue *context.RealUe,
	msg common.InterfaceMessage) {

	ue.Log.Traceln("Sending", common.GetEvtString(msg.GetEventType()), "to SimUe")
	ue.WriteSimUeChan <- msg
}
