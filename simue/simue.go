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
	uemsg.Interface = common.UU_INTERFACE
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

	ue.Log.Infoln("Handling event:", msg.GetEventType(), "from interface:",
		msg.GetInterfaceType())

	switch msg.GetInterfaceType() {

	case common.UU_INTERFACE:
		uuMsg := msg.(*common.UuMessage)

		switch uuMsg.Event {
		case common.REG_REQUEST_EVENT:
			err = HandleRegReqEvent(ue, uuMsg)
		case common.AUTH_REQUEST_EVENT:
			err = HandleAuthRequestEvent(ue, uuMsg)
		case common.AUTH_RESPONSE_EVENT:
			err = HandleAuthResponseEvent(ue, uuMsg)
		case common.SEC_MOD_COMMAND_EVENT:
			err = HandleSecModCommandEvent(ue, uuMsg)
		case common.SEC_MOD_COMPLETE_EVENT:
			err = HandleSecModCompleteEvent(ue, uuMsg)
		case common.REG_ACCEPT_EVENT:
			err = HandleRegAcceptEvent(ue, uuMsg)
		case common.REG_COMPLETE_EVENT:
			err = HandleRegCompleteEvent(ue, uuMsg)
		case common.PDU_SESS_EST_REQUEST_EVENT:
			err = HandlePduSessEstRequestEvent(ue, uuMsg)
		case common.PDU_SESS_EST_ACCEPT_EVENT:
			err = HandlePduSessEstAcceptEvent(ue, uuMsg)
		case common.DL_INFO_TRANSFER_EVENT:
			err = HandleDlInfoTransferEvent(ue, uuMsg)
		case common.DATA_BEARER_SETUP_REQUEST_EVENT:
			err = HandleDataBearerSetupRequestEvent(ue, uuMsg)
		case common.DATA_BEARER_SETUP_RESPONSE_EVENT:
			err = HandleDataBearerSetupResponseEvent(ue, uuMsg)
		case common.DATA_PKT_GEN_SUCCESS_EVENT:
			err = HandleDataPktGenSuccessEvent(ue, uuMsg)
		case common.DATA_PKT_GEN_FAILURE_EVENT:
			err = HandleDataPktGenFailureEvent(ue, uuMsg)
		default:
			ue.Log.Infoln("Event", uuMsg.Event, "is not supported")
		}

	case common.PROFILE_SIMUE_INTERFACE:
		profileMsg := msg.(*common.ProfileMessage)

		switch profileMsg.Event {
		case common.PROFILE_START_EVENT:
			err = HandleProfileStartEvent(ue, profileMsg)
		default:
			ue.Log.Infoln("Event", profileMsg.Event, "is not supported")
		}

	default:
		ue.Log.Infoln("Interface", msg.GetInterfaceType(), "is not supported")
	}

	if err != nil {
		ue.Log.Errorln("Failed to process event:", msg.GetEventType(), "Error:", err)
	}

	return err
}

func SendToRealUe(ue *context.SimUe, msg *common.UuMessage) {
	ue.Log.Infoln("Sending", msg.Event, "to RealUe")
	ue.WriteRealUeChan <- msg
}

func SendToGnbUe(ue *context.SimUe, msg *common.UuMessage) {
	ue.Log.Infoln("Sending", msg.Event, "to GnbUe")
	ue.WriteGnbUeChan <- msg
}

func SendToProfile(ue *context.SimUe, event common.EventType, errMsg error) {
	ue.Log.Infoln("Sending event", event, "to Profile routine")
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Supi = ue.Supi
	msg.Proc = ue.Procedure
	msg.ErrorMsg = errMsg
	ue.WriteProfileChan <- msg
}
