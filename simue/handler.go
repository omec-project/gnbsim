// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package simue

import (
	"gnbsim/common"
	"gnbsim/simue/context"
)

func HandleProfileStartEvent(ue *context.SimUe, msg *common.ProfileMessage) (err error) {
	ue.Log.Traceln("Handling Profile Start Event")
	ue.Procedure = ue.ProfileCtx.GetFirstProcedure()
	ue.Log.Infoln("Updated procedure to", ue.Procedure)
	HandleProcedure(ue)
	return nil
}

func HandleRegReqEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Registration Request Event")
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Registration Request Event to GnbUe")
	return nil
}

func HandleAuthRequestEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Authentication Request Event")
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

func HandleAuthResponseEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Authentication Response Event")

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

func HandleSecModCommandEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Security Mode Command Event")
	// TODO: Should check if SecModCommandEvent event is expected

	nextEvent, err := ue.ProfileCtx.GetNextEvent(msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleSecModCompleteEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Security Mode Complete Event")
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

func HandleRegAcceptEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Registration Accept Event")
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

func HandleRegCompleteEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Registration Complete Event")
	err = ue.ProfileCtx.CheckCurrentEvent(common.REG_ACCEPT_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent UL Information Transfer[Registration Complete] Event to GnbUe")
	nextProcedure := ue.ProfileCtx.GetNextProcedure(ue.Procedure)
	if nextProcedure != 0 {
		ue.Procedure = nextProcedure
		ue.Log.Infoln("Updated procedure to", ue.Procedure)
		HandleProcedure(ue)
	} else {
		SendToProfile(ue, common.PROFILE_PASS_EVENT, nil)
		ue.Log.Traceln("Sent Profile Pass Event to Profile routine")
	}
	return nil
}

func HandlePduSessEstRequestEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling PDU Session Establishment Request Event")
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent PDU Session Establishment Request to GnbUe")
	return nil
}

func HandlePduSessEstAcceptEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling PDU Session Establishment Accept Event")
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

	nextProcedure := ue.ProfileCtx.GetNextProcedure(ue.Procedure)
	if nextProcedure != 0 {
		ue.Procedure = nextProcedure
		ue.Log.Infoln("Updated procedure to", ue.Procedure)
		HandleProcedure(ue)
	} else {
		SendToProfile(ue, common.PROFILE_PASS_EVENT, nil)
		ue.Log.Traceln("Sent Profile Pass Event to Profile routine")
	}
	return nil
}

func HandleDlInfoTransferEvent(ue *context.SimUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling DL Information Transfer Event")
	SendToRealUe(ue, msg)
	ue.Log.Traceln("Sent DL Information Event to RealUE")
	return nil
}

func HandleDataBearerSetupRequestEvent(ue *context.SimUe,
	msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Data Bearer Setup Request Event")
	SendToRealUe(ue, msg)
	ue.Log.Traceln("Sent Data Bearer Setup Request to RealUE")
	return nil
}

func HandleProcedure(ue *context.SimUe) {
	switch ue.Procedure {
	case common.REGISTRATION_PROCEDURE:
		ue.Log.Traceln("Initiating Registration Procedure")
		msg := &common.UuMessage{}
		msg.Event = common.REG_REQUEST_EVENT
		SendToRealUe(ue, msg)
		ue.Log.Traceln("Sent Registration Request Event to RealUe")
	case common.PDU_SESSION_ESTABLISHMENT_PROCEDURE:
		ue.Log.Traceln("Initiating UE Requested PDU Session Establishment Procedure")
		msg := &common.UuMessage{}
		msg.Event = common.PDU_SESS_EST_REQUEST_EVENT
		SendToRealUe(ue, msg)
		ue.Log.Traceln("Sent PDU Session Establishment Request Event to RealUe")
	}
}
