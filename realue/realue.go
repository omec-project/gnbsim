// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package realue

import (
	"github.com/omec-project/gnbsim/common"
	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/gnbsim/util/test"
)

func Init(ue *realuectx.RealUe) {
	ue.AuthenticationSubs = test.GetAuthSubscription(ue.Key, ue.Opc, "", ue.SeqNum)

	err := HandleEvents(ue)
	if err != nil {
		ue.Log.Infoln("failed to handle events:", err)
	}
}

func HandleEvents(ue *realuectx.RealUe) (err error) {
	for msg := range ue.ReadChan {
		event := msg.GetEventType()
		ue.Log.Infoln("Handling:", event)

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
		case common.PDU_SESS_REL_REQUEST_EVENT:
			err = HandlePduSessReleaseRequestEvent(ue, msg)
		case common.PDU_SESS_REL_COMPLETE_EVENT:
			err = HandlePduSessReleaseCompleteEvent(ue, msg)
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
		case common.DEREG_ACCEPT_UE_TERM_EVENT:
			err = HandleNwDeregAcceptEvent(ue, msg)
		case common.ERROR_EVENT:
			err = HandleErrorEvent(ue, msg)
		case common.QUIT_EVENT:
			err = HandleQuitEvent(ue, msg)
			if err != nil {
				ue.Log.Warnln("failed to handle quiet event", err)
			}
			return nil
		default:
			ue.Log.Warnln("Event", event, "is not supported")
		}

		if err != nil {
			ue.Log.Errorln("real ue failed:", event, ":", err)
			msg := &common.UeMessage{}
			msg.Error = err
			msg.Event = common.ERROR_EVENT
			err = HandleErrorEvent(ue, msg)
		}
	}

	return nil
}

func formUuMessage(event common.EventType, nasPdu []byte, id uint64) *common.UuMessage {
	msg := &common.UuMessage{}
	msg.Event = event
	msg.NasPdus = append(msg.NasPdus, nasPdu)
	msg.Id = id
	return msg
}

func SendToSimUe(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) {
	ue.Log.Traceln("Sending", msg.GetEventType(), "to SimUe")
	ue.WriteSimUeChan <- msg
}
