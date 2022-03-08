// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"fmt"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/gnodeb"
	"github.com/omec-project/gnbsim/realue"
	"github.com/omec-project/gnbsim/simue/context"
)

func Init(simUe *context.SimUe, wg *sync.WaitGroup) {

	err := ConnectToGnb(simUe)
	if err != nil {
		err = fmt.Errorf("failed to connect to gnodeb:", err)
		SendToProfile(simUe, common.PROFILE_FAIL_EVENT, err)
		simUe.Log.Infoln("Sent Profile Fail Event to Profile routine")
		return
	}

	simUe.WaitGrp.Add(1)
	go realue.Init(simUe.RealUe, &simUe.WaitGrp)

	HandleEvents(simUe)
	wg.Done()
}

func ConnectToGnb(simUe *context.SimUe) error {
	uemsg := common.UuMessage{}
	uemsg.Event = common.CONNECTION_REQUEST_EVENT
	uemsg.CommChan = simUe.ReadChan
	uemsg.Supi = simUe.Supi

	var err error
	gNb := simUe.GnB
	simUe.WriteGnbUeChan, err = gnodeb.RequestConnection(gNb, &uemsg)
	if err != nil {
		return err
	}

	simUe.Log.Infof("Connected to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
		gNb.GnbN2Ip, gNb.GnbN2Port)
	return nil
}

func HandleEvents(ue *context.SimUe) {
	var err error
	for msg := range ue.ReadChan {
		event := msg.GetEventType()
		ue.Log.Infoln("Handling event:", event)

		switch event {
		case common.REG_REQUEST_EVENT:
			err = HandleRegRequestEvent(ue, msg)
		case common.REG_REJECT_EVENT:
			err = HandleRegRejectEvent(ue, msg)
		case common.AUTH_REQUEST_EVENT:
			err = HandleAuthRequestEvent(ue, msg)
		case common.AUTH_RESPONSE_EVENT:
			err = HandleAuthResponseEvent(ue, msg)
		case common.SEC_MOD_COMMAND_EVENT:
			err = HandleSecModCommandEvent(ue, msg)
		case common.SEC_MOD_COMPLETE_EVENT:
			err = HandleSecModCompleteEvent(ue, msg)
		case common.REG_ACCEPT_EVENT:
			err = HandleRegAcceptEvent(ue, msg)
		case common.REG_COMPLETE_EVENT:
			err = HandleRegCompleteEvent(ue, msg)
		case common.DEREG_REQUEST_UE_ORIG_EVENT:
			err = HandleDeregRequestEvent(ue, msg)
		case common.DEREG_ACCEPT_UE_ORIG_EVENT:
			err = HandleDeregAcceptEvent(ue, msg)
		case common.PDU_SESS_EST_REQUEST_EVENT:
			err = HandlePduSessEstRequestEvent(ue, msg)
		case common.PDU_SESS_EST_ACCEPT_EVENT:
			err = HandlePduSessEstAcceptEvent(ue, msg)
		case common.PDU_SESS_EST_REJECT_EVENT:
			err = HandlePduSessEstRejectEvent(ue, msg)
		case common.DL_INFO_TRANSFER_EVENT:
			err = HandleDlInfoTransferEvent(ue, msg)
		case common.DATA_BEARER_SETUP_REQUEST_EVENT:
			err = HandleDataBearerSetupRequestEvent(ue, msg)
		case common.DATA_BEARER_SETUP_RESPONSE_EVENT:
			err = HandleDataBearerSetupResponseEvent(ue, msg)
		case common.DATA_PKT_GEN_SUCCESS_EVENT:
			err = HandleDataPktGenSuccessEvent(ue, msg)
		case common.DATA_PKT_GEN_FAILURE_EVENT:
			err = HandleDataPktGenFailureEvent(ue, msg)
		case common.SERVICE_REQUEST_EVENT:
			err = HandleServiceRequestEvent(ue, msg)
		case common.SERVICE_ACCEPT_EVENT:
			err = HandleServiceAcceptEvent(ue, msg)
		case common.PROFILE_START_EVENT:
			err = HandleProfileStartEvent(ue, msg)
		case common.CONNECTION_RELEASE_REQUEST_EVENT:
			err = HandleConnectionReleaseRequestEvent(ue, msg)
		case common.ERROR_EVENT:
			HandleErrorEvent(ue, msg)
			return
		case common.QUIT_EVENT:
			HandleQuitEvent(ue, msg)
			return
		default:
			ue.Log.Infoln("Event:", event, "is not supported")
		}

		if err != nil {
			ue.Log.Errorln("Failed to handle event:", event, "Error:", err)
			msg := &common.UeMessage{}
			msg.Error = err
			err = nil
			msg.Event = common.ERROR_EVENT
			HandleErrorEvent(ue, msg)
		}
	}

	return
}

func SendToRealUe(ue *context.SimUe, msg common.InterfaceMessage) {
	ue.Log.Traceln("Sending", msg.GetEventType(), "to RealUe")
	ue.WriteRealUeChan <- msg
}

func SendToGnbUe(ue *context.SimUe, msg common.InterfaceMessage) {
	ue.Log.Traceln("Sending", msg.GetEventType(), "to GnbUe")
	ue.WriteGnbUeChan <- msg
}

func SendToProfile(ue *context.SimUe, event common.EventType, errMsg error) {
	ue.Log.Traceln("Sending", event, "to Profile routine")
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Supi = ue.Supi
	msg.Proc = ue.Procedure
	msg.Error = errMsg
	ue.WriteProfileChan <- msg
}
