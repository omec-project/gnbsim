// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package simue

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/gnodeb"
	"gnbsim/realue"
	"gnbsim/simue/context"
)

func Init(simUe *context.SimUe) {
	var err error
	defer func(err error) {
		if err != nil {
			SendToProfile(simUe, common.PROFILE_FAIL_EVENT, err)
			simUe.Log.Infoln("Sent Profile Fail Event to Profile routine")
		}
	}(err)

	go realue.Init(simUe.RealUe)
	err = ConnectToGnb(simUe)
	if err != nil {
		simUe.Log.Errorln("ConnectToGnb returned:", err)
		err = fmt.Errorf("failed to connect to gnodeb")
		return
	}

	for msg := range simUe.ReadChan {
		err := HandleEvent(simUe, msg)
		if err != nil {
			simUe.Log.Errorln("HandleEvent returned:", err)
			err = fmt.Errorf("failed to handle received event")
			return
		}
	}
}

func ConnectToGnb(simUe *context.SimUe) error {
	uemsg := common.UuMessage{}
	uemsg.Event = common.CONNECT_REQUEST_EVENT
	uemsg.CommChan = simUe.ReadChan
	uemsg.Supi = simUe.Supi

	var err error
	gNb := simUe.GnB
	simUe.WriteGnbUeChan, err = gnodeb.RequestConnection(gNb, &uemsg)
	if err != nil {
		return fmt.Errorf("failed to establish connection with gnb, err: %v", err)
	}

	simUe.Log.Infof("Connected to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
		gNb.GnbN2Ip, gNb.GnbN2Port)
	return nil
}

func HandleEvent(ue *context.SimUe, msg common.InterfaceMessage) (err error) {
	if msg == nil {
		return fmt.Errorf("empty message received")
	}

	ue.Log.Infoln("Handling event:", msg.GetEventType())

	switch msg.GetEventType() {
	case common.REG_REQUEST_EVENT:
		err = HandleRegReqEvent(ue, msg)
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
	case common.PDU_SESS_EST_REQUEST_EVENT:
		err = HandlePduSessEstRequestEvent(ue, msg)
	case common.PDU_SESS_EST_ACCEPT_EVENT:
		err = HandlePduSessEstAcceptEvent(ue, msg)
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
	case common.PROFILE_START_EVENT:
		err = HandleProfileStartEvent(ue, msg)
	default:
		ue.Log.Infoln("Event", msg.GetEventType(), "is not supported")
	}

	if err != nil {
		ue.Log.Errorln("Failed to process event:", msg.GetEventType(), "Error:", err)
	}

	return err
}

func SendToRealUe(ue *context.SimUe, msg common.InterfaceMessage) {
	ue.Log.Infoln("Sending", msg.GetEventType(), "to RealUe")
	ue.WriteRealUeChan <- msg
}

func SendToGnbUe(ue *context.SimUe, msg common.InterfaceMessage) {
	ue.Log.Infoln("Sending", msg.GetEventType(), "to GnbUe")
	ue.WriteGnbUeChan <- msg
}

func SendToProfile(ue *context.SimUe, event common.EventType, errMsg error) {
	ue.Log.Infoln("Sending event", event, "to Profile routine")
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Supi = ue.Supi
	msg.Proc = ue.Procedure
	msg.Error = errMsg
	ue.WriteProfileChan <- msg
}
