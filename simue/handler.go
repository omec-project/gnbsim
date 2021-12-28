// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package simue

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/simue/context"
	"time"
)

func HandleProfileStartEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Profile Start Event")

	ue.Procedure = ue.ProfileCtx.GetFirstProcedure()
	ue.Log.Infoln("Updated procedure to", ue.Procedure)
	HandleProcedure(ue)
	return nil
}

func HandleRegRequestEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Registration Request Event")

	SendToGnbUe(ue, intfcMsg)
	ue.Log.Traceln("Sent Registration Request Event to GnbUe")
	return nil
}

func HandleAuthRequestEvent(ue *context.SimUe,
	intfMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Authentication Request Event")

	msg := intfMsg.(*common.UeMessage)
	// checking as per profile if Authentication Request Message is expected
	// from 5G Core against Registration Request message sent by RealUE
	err = ue.ProfileCtx.CheckCurrentEvent(common.REG_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}
	nextEvent, err := ue.ProfileCtx.GetNextEvent(msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("Next Event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleAuthResponseEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Authentication Response Event")

	msg := intfcMsg.(*common.UuMessage)
	// Checking if RealUe has sent expected message as per profile against
	// Authentication Request message recevied from 5G Core
	err = ue.ProfileCtx.CheckCurrentEvent(common.AUTH_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent UL Information Transfer[Authentication Response] Event to GnbUe")
	return nil
}

func HandleSecModCommandEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Security Mode Command Event")
	// TODO: Should check if SecModCommandEvent event is expected

	msg := intfcMsg.(*common.UeMessage)
	nextEvent, err := ue.ProfileCtx.GetNextEvent(msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleSecModCompleteEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Security Mode Complete Event")

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(common.SEC_MOD_COMMAND_EVENT,
		msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent UL Information Transfer[Security Mode Complete] Event to GnbUe")
	return nil
}

func HandleRegAcceptEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Registration Accept Event")

	msg := intfcMsg.(*common.UeMessage)
	// TODO: Should check if Registration Accept event is expected
	nextEvent, err := ue.ProfileCtx.GetNextEvent(msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleRegCompleteEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Registration Complete Event")

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(common.REG_ACCEPT_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent UL Information Transfer[Registration Complete] Event to GnbUe")

	ChangeProcedure(ue)
	return nil
}

func HandleDeregRequestEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling UE Originated Deregistration Request Event")

	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent UL Information Transfer[Deregistration Request] Event to GnbUe")

	return nil
}

func HandleDeregAcceptEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling UE Originated Deregistration Accept Event")

	return nil
}

// HandleCtxRelAckEvent handler is called upon receiving acknowledgement
// from gNB for releasing the UE context. This ensures SimUE that the UE
// originated Deregisteration procedure followed by AN Release procedure is
// completed
func HandleCtxRelAckEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling UE context release acknowledgement from gNB")

	ChangeProcedure(ue)
	return nil
}

func HandlePduSessEstRequestEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling PDU Session Establishment Request Event")

	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent PDU Session Establishment Request to GnbUe")
	return nil
}

func HandlePduSessEstAcceptEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling PDU Session Establishment Accept Event")

	msg := intfcMsg.(*common.UeMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(common.PDU_SESS_EST_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}
	nextEvent, err := ue.ProfileCtx.GetNextEvent(msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("Next Event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleDlInfoTransferEvent(ue *context.SimUe,
	msg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling DL Information Transfer Event")

	SendToRealUe(ue, msg)
	ue.Log.Traceln("Sent DL Information Event to RealUE")
	return nil
}

func HandleDataBearerSetupRequestEvent(ue *context.SimUe,
	msg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Data Bearer Setup Request Event")

	SendToRealUe(ue, msg)
	ue.Log.Traceln("Sent Data Bearer Setup Request to RealUE")
	return nil
}

func HandleDataBearerSetupResponseEvent(ue *context.SimUe,
	msg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Data Bearer Setup Response Event")

	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Data Bearer Setup Response to RealUE")

	ChangeProcedure(ue)
	return nil
}

func HandleDataPktGenSuccessEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Data Packet Generation Success Event")

	ChangeProcedure(ue)
	return nil
}

func HandleDataPktGenFailureEvent(ue *context.SimUe,
	msg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Data Packet Generation Failure Event")

	SendToProfile(ue, common.PROFILE_FAIL_EVENT, msg.GetErrorMsg())
	return nil
}

func HandleServiceRequestEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Service Request Event")

	err = ConnectToGnb(ue)
	if err != nil {
		return fmt.Errorf("failed to connect gnb:", err)
	}

	SendToGnbUe(ue, intfcMsg)

	ue.Log.Traceln("Sent Service Request Event to GnbUe")
	return nil
}

func HandleServiceAcceptEvent(ue *context.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Service Request Accept Event")

	err = ue.ProfileCtx.CheckCurrentEvent(common.SERVICE_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	//ChangeProcedure(ue)

	return nil
}

func ChangeProcedure(ue *context.SimUe) {
	nextProcedure := ue.ProfileCtx.GetNextProcedure(ue.Procedure)
	if nextProcedure != 0 {
		ue.Procedure = nextProcedure
		ue.Log.Infoln("Updated procedure to", ue.Procedure)
		HandleProcedure(ue)
	} else {
		SendToProfile(ue, common.PROFILE_PASS_EVENT, nil)
		ue.Log.Traceln("Sent Profile Pass Event to Profile routine")
	}
}

func HandleProcedure(ue *context.SimUe) {
	switch ue.Procedure {
	case common.REGISTRATION_PROCEDURE:
		ue.Log.Infoln("Initiating Registration Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.REG_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.PDU_SESSION_ESTABLISHMENT_PROCEDURE:
		ue.Log.Infoln("Initiating UE Requested PDU Session Establishment Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.PDU_SESS_EST_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.USER_DATA_PKT_GENERATION_PROCEDURE:
		ue.Log.Infoln("Initiating User Data Packet Generation Procedure")
		msg := &common.UeMessage{}
		msg.UserDataPktCount = ue.ProfileCtx.DataPktCount
		msg.Event = common.DATA_PKT_GEN_REQUEST_EVENT

		time.Sleep(500 * time.Millisecond)
		/* TODO: Solve timing issue. Currently UE may start sending user data
		 * before gnb has successfuly sent PDU Session Resource Setup Response
		 * or before 5g core has processed it
		 */
		ue.Log.Infoln("Please wait, initiating uplink user data in 3 seconds ...")
		time.Sleep(3 * time.Second)

		SendToRealUe(ue, msg)
	case common.UE_INITIATED_DEREGISTRATION_PROCEDURE:
		ue.Log.Infoln("Initiating UE Initiated Deregistration Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.DEREG_REQUEST_UE_ORIG_EVENT
		SendToRealUe(ue, msg)
	case common.AN_RELEASE_PROCEDURE:
		ue.Log.Infoln("Initiating AN Release Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.RAN_CONNECTION_RELEASE_EVENT
		SendToGnbUe(ue, msg)
	case common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE:
		ue.Log.Infoln("Initiating UE Triggered Service Request Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.SERVICE_REQUEST_EVENT
		SendToRealUe(ue, msg)
	}
}
