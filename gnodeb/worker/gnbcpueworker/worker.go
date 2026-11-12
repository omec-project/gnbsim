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

// nextEvent returns the next event for the context. Queued events come first, so that a response
// that was already handed over is still processed before the context is torn down; a quit
// request is returned as a QUIT_EVENT once the queue is empty.
func nextEvent(gnbue *gnbctx.GnbCpUe) (common.InterfaceMessage, bool) {
	select {
	case msg, ok := <-gnbue.ReadChan:
		return msg, ok
	default:
	}
	select {
	case msg, ok := <-gnbue.ReadChan:
		return msg, ok
	case <-gnbue.QuitRequested():
		return &common.DefaultMessage{Event: common.QUIT_EVENT}, true
	}
}

func HandleEvents(gnbue *gnbctx.GnbCpUe) (err error) {
	if gnbue.Done != nil {
		defer close(gnbue.Done)
	}
	for {
		msg, ok := nextEvent(gnbue)
		if !ok {
			return nil
		}
		evt := msg.GetEventType()
		gnbue.Log.Infoln("Handling event:", evt)

		handleEvent(gnbue, msg, evt)

		// the context has been torn down and unregistered, there is nothing left to serve
		if evt == common.QUIT_EVENT {
			return nil
		}

		// TODO: Need to return and handle errors from handlers
	}
}

// handleEvent dispatches one event. A panic caused by a malformed message (the NGAP decoder
// does not enforce mandatory IEs) is contained so that it neither kills the process nor the
// event loop of this UE.
func handleEvent(gnbue *gnbctx.GnbCpUe, msg common.InterfaceMessage, evt common.EventType) {
	defer func() {
		if r := recover(); r != nil {
			gnbue.Log.Errorf("panic while handling event %v: %v", evt, r)
		}
	}()

	switch evt {
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
	case common.PDU_SESS_RESOURCE_MODIFY_REQUEST_EVENT:
		HandlePduSessResourceModifyRequest(gnbue, msg)
	case common.UE_CTX_RELEASE_COMMAND_EVENT:
		HandleUeCtxReleaseCommand(gnbue, msg)
	case common.TRIGGER_AN_RELEASE_EVENT:
		HandleRanConnectionRelease(gnbue, msg)
	case common.TRIGGER_HO_EVENT:
		HandleTriggerHandover(gnbue, msg)
	case common.HO_REQUEST_EVENT:
		HandleHandoverRequest(gnbue, msg)
	case common.HO_COMMAND_EVENT:
		HandleHandoverCommand(gnbue, msg)
	case common.HO_NOTIFY_EVENT:
		HandleHandoverNotify(gnbue, msg)
	case common.QUIT_EVENT:
		HandleQuitEvent(gnbue, msg)
	default:
		gnbue.Log.Infoln("event", evt, "is not supported")
	}
}

// sendToSimUe delivers a message to the SimUe. A SimUe that has stopped reading, for instance
// because it gave up on a timeout, must not keep this context from seeing its quit request and
// terminating, so the send gives way to it.
func sendToSimUe(gnbue *gnbctx.GnbCpUe, msg common.InterfaceMessage) {
	// delivered if it can be, whether or not a quit has been requested: the choice below between
	// two ready cases is random, and a message that could have gone out would be lost by chance
	select {
	case gnbue.WriteUeChan <- msg:
		return
	default:
	}
	select {
	case gnbue.WriteUeChan <- msg:
	case <-gnbue.QuitRequested():
		gnbue.Log.Debugln("dropped", msg.GetEventType(), "for the SimUe, the context is quitting")
	}
}

func SendToUe(gnbue *gnbctx.GnbCpUe, event common.EventType, nasPdus common.NasPduList, id uint64) {
	gnbue.Log.Debugln("sending event", event, "to SimUe. Id:", id)
	uemsg := common.UuMessage{}
	uemsg.Id = id
	uemsg.Event = event
	uemsg.NasPdus = nasPdus
	sendToSimUe(gnbue, &uemsg)
}
