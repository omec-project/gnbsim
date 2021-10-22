// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbcpueworker

import (
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"log"
)

func Init(gnbue *context.GnbCpUe) {
	for {
		select {
		case msg := <-gnbue.ReadChan:
			err := HandleMessage(gnbue, msg)
			if err != nil {
				log.Println(err)
			}
		case <-gnbue.Gnb.Quit:
			return
		}
	}
}

func HandleMessage(gnbue *context.GnbCpUe, msg common.InterfaceMessage) (err error) {

	gnbue.Log.Infoln("Handling event:", msg.GetEventType())

	switch msg.GetEventType() {
	case common.CONNECT_REQUEST_EVENT:
		HandleConnectRequest(gnbue, msg)
	case common.REG_REQUEST_EVENT:
		HandleInitialUEMessage(gnbue, msg)
	case common.UL_INFO_TRANSFER_EVENT:
		HandleUlInfoTransfer(gnbue, msg)
	case common.DATA_BEARER_SETUP_RESPONSE_EVENT:
		HandleDataBearerSetupResponse(gnbue, msg)
	case common.DOWNLINK_NAS_TRANSPORT_EVENT:
		HandleDownlinkNasTransport(gnbue, msg)
	case common.INITIAL_CONTEXT_SETUP_REQUEST_EVENT:
		HandleInitialContextSetupRequest(gnbue, msg)
	case common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT:
		HandlePduSessResourceSetupRequest(gnbue, msg)
	default:
		gnbue.Log.Infoln("Event", msg.GetEventType(), "is not supported")
	}
	return nil
}

func SendToUe(gnbue *context.GnbCpUe, event common.EventType, nasPdus common.NasPduList) {
	gnbue.Log.Infoln("Sending event", event, "to SimUe")
	uemsg := common.UuMessage{}
	uemsg.Event = event
	uemsg.NasPdus = nasPdus
	gnbue.WriteUeChan <- &uemsg
}
