// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
)

func Init(gnbue *gnbctx.GnbCpUe) {
	err := HandleEvents(gnbue)
	if err != nil {
		gnbue.Log.Infoln("failed to handle event", err)
	}
}

func HandleEvents(gnbue *gnbctx.GnbCpUe) (err error) {
	for msg := range gnbue.ReadChan {
		evt := msg.GetEventType()
		gnbue.Log.Infoln("Handling event:", evt)

		switch msg.GetEventType() {
		case common.CONNECTION_REQUEST_EVENT:
			HandleConnectRequest(gnbue, msg)
		case common.REG_REQUEST_EVENT, common.SERVICE_REQUEST_EVENT:
			HandleInitialUEMessage(gnbue, msg)
		case common.UL_INFO_TRANSFER_EVENT:
			HandleUlInfoTransfer(gnbue, msg)
		case common.DATA_BEARER_SETUP_RESPONSE_EVENT:
			HandleDataBearerSetupResponse(gnbue, msg)
		case common.DOWNLINK_NAS_TRANSPORT_EVENT:
			HandleDownlinkNasTransport(gnbue, msg)
		case common.INITIAL_CTX_SETUP_REQUEST_EVENT:
			HandleInitialContextSetupRequest(gnbue, msg)
		case common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT:
			HandlePduSessResourceSetupRequest(gnbue, msg)
		case common.PDU_SESS_RESOURCE_RELEASE_COMMAND_EVENT:
			HandlePduSessResourceReleaseCommand(gnbue, msg)
		case common.UE_CTX_RELEASE_COMMAND_EVENT:
			HandleUeCtxReleaseCommand(gnbue, msg)
		case common.TRIGGER_AN_RELEASE_EVENT:
			HandleRanConnectionRelease(gnbue, msg)
		case common.QUIT_EVENT:
			HandleQuitEvent(gnbue, msg)
		default:
			gnbue.Log.Infoln("Event", evt, "is not supported")
		}

		// TODO: Need to return and handle errors from handlers
	}
	return nil
}

func SendToUe(gnbue *gnbctx.GnbCpUe, event common.EventType, nasPdus common.NasPduList, id uint64) {
	gnbue.Log.Traceln("Sending event", event, "to SimUe. Id:", id)
	uemsg := common.UuMessage{}
	uemsg.Id = id
	uemsg.Event = event
	uemsg.NasPdus = nasPdus
	gnbue.WriteUeChan <- &uemsg
}
