// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	simuectx "github.com/omec-project/gnbsim/simue/context"
)

func HandleProfileStartEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	if len(ue.ProfileCtx.PIterations) > 0 {
		ue.CurrentItr = ue.ProfileCtx.StartIteration
		pItr := ue.ProfileCtx.PIterations[ue.CurrentItr]
		ue.Procedure = pItr.ProcMap[1]
		ue.CurrentProcIndex = 1
		ue.Repeat = pItr.Repeat
	} else {
		ue.Procedure = ue.ProfileCtx.GetFirstProcedure()
	}
	ue.Log.Infoln("Updated procedure to", ue.Procedure)
	HandleProcedure(ue)
	return nil
}

func HandleProfileContinueEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	if len(ue.ProfileCtx.PIterations) > 0 {
		pItr := ue.ProfileCtx.PIterations[ue.CurrentItr]
		ue.Log.Infoln("HandleProfileContinueEvent CurrentItr ", ue.CurrentItr)
		ue.Procedure = pItr.ProcMap[1]
		ue.CurrentProcIndex = 1
		ue.Repeat = pItr.Repeat
	} else {
		ue.Procedure = ue.ProfileCtx.GetFirstProcedure()
	}
	ue.Log.Infoln("Updated procedure to", ue.Procedure)
	HandleProcedure(ue)
	return nil
}

func HandleRegRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	SendToGnbUe(ue, intfcMsg)
	return nil
}

func HandleRegRejectEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.REG_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	return nil
}

func HandleAuthRequestEvent(ue *simuectx.SimUe,
	intfMsg common.InterfaceMessage) (err error) {

	msg := intfMsg.(*common.UeMessage)
	// checking as per profile if Authentication Request Message is expected
	// from 5G Core against Registration Request message sent by RealUE
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.REG_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}
	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("Next Event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleAuthResponseEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UuMessage)
	// Checking if RealUe has sent expected message as per profile against
	// Authentication Request message recevied from 5G Core
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.AUTH_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sending Authentication Response to the network")
	return nil
}

func HandleSecModCommandEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	// TODO: Should check if SecModCommandEvent event is expected

	msg := intfcMsg.(*common.UeMessage)
	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleSecModCompleteEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Security Mode Complete Event")

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.SEC_MOD_COMMAND_EVENT,
		msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Security Mode Complete to the network")
	return nil
}

func HandleRegAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UeMessage)
	// TODO: Should check if Registration Accept event is expected
	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleRegCompleteEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.REG_ACCEPT_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Registration Complete to the network")

	//Current Procedure is complete. Move to next one
	ChangeProcedure(ue)
	return nil
}

func HandleDeregRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Deregistration Request to the network")

	return nil
}

func HandleDeregAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	return nil
}

func HandlePduSessEstRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

func HandlePduSessEstAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UeMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.PDU_SESS_EST_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}
	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("Next Event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandlePduSessEstRejectEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.PDU_SESS_EST_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	return nil
}

func HandlePduSessReleaseRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

func HandlePduSessReleaseCommandEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UeMessage)
	if ue.Procedure == common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE {
		err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.PDU_SESS_REL_REQUEST_EVENT, msg.Event)
		if err != nil {
			ue.Log.Errorln("CheckCurrentEvent returned:", err)
			return err
		}
	}
	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("Next Event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandlePduSessReleaseCompleteEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

func HandleDlInfoTransferEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage) (err error) {

	SendToRealUe(ue, msg)
	return nil
}

func HandleDataBearerSetupRequestEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage) (err error) {

	SendToRealUe(ue, msg)
	return nil
}

func HandleDataBearerSetupResponseEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage) (err error) {

	SendToGnbUe(ue, msg)

	//Current Procedure is complete. Move to next one
	ChangeProcedure(ue)
	return nil
}

func HandleDataBearerReleaseRequestEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage) (err error) {
	// This event is sent by gNB component after it has sent
	// PDU Session Resource Release Complete over N2, However the PDU Sesson
	// routines in the RealUE will be terminated while processing PDU Session
	// Release Complete which will also release the communication links
	// (go channels) with the gNB
	//Current Procedure is complete. Move to next one
	ChangeProcedure(ue)
	return nil
}

func HandleDataPktGenSuccessEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	//Current Procedure is complete. Move to next one
	ChangeProcedure(ue)
	return nil
}

func HandleDataPktGenFailureEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage) (err error) {

	SendToProfile(ue, common.PROFILE_FAIL_EVENT, msg.GetErrorMsg())
	return nil
}

func HandleServiceRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	err = ConnectToGnb(ue)
	if err != nil {
		return fmt.Errorf("failed to connect gnb %v:", err)
	}

	SendToGnbUe(ue, intfcMsg)

	ue.Log.Traceln("Sent Service Request Event to the network")
	return nil
}

func HandleServiceAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.SERVICE_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	return nil
}

func HandleConnectionReleaseRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {
	msg := intfcMsg.(*common.UuMessage)

	if ue.Procedure == common.AN_RELEASE_PROCEDURE {
		err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.TRIGGER_AN_RELEASE_EVENT,
			common.CONNECTION_RELEASE_REQUEST_EVENT)
		if err != nil {
			return err
		}
	}

	ue.WriteGnbUeChan = nil

	if msg.TriggeringEvent == common.DEREG_REQUEST_UE_ORIG_EVENT {
		msg := &common.UeMessage{}
		msg.Event = common.QUIT_EVENT
		ue.ReadChan <- msg
		if len(ue.ProfileCtx.Iterations) > 0 {
			itp := ue.ProfileCtx.PIterations[ue.CurrentItr]
			nextItr := itp.NextItr
			if nextItr != "quit" {
				nItr := ue.ProfileCtx.PIterations[nextItr]
				SendToProfile(ue, common.PROFILE_CONT_EVENT, fmt.Errorf("%v", nItr.Name))
				return nil
			}
		}
		// Nothing else to execute. Tell profile we are done.
		SendToProfile(ue, common.PROFILE_PASS_EVENT, nil)
		return nil
	}
	SendToRealUe(ue, msg)
	//Current Procedure is complete. Move to next one
	ChangeProcedure(ue)

	return nil
}

func HandleNwDeregRequestEvent(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage) (err error) {

	msg := intfcMsg.(*common.UeMessage)

	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("Next Event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)

	return nil
}

func HandleNwDeregAcceptEvent(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage) (err error) {

	ue.Log.Traceln("Handling Dereg Accept Event")

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.DEREG_REQUEST_UE_TERM_EVENT,
		msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Dereg Accept to the network")
	return nil
}

func HandleErrorEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage) (err error) {

	SendToProfile(ue, common.PROFILE_FAIL_EVENT, intfcMsg.GetErrorMsg())

	msg := &common.UuMessage{}
	msg.Event = common.QUIT_EVENT
	HandleQuitEvent(ue, msg)
	return nil
}

func HandleQuitEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage) (err error) {
	if ue.WriteGnbUeChan != nil {
		SendToGnbUe(ue, msg)
	}
	SendToRealUe(ue, msg)
	ue.WriteRealUeChan = nil
	ue.WaitGrp.Wait()
	ue.Log.Infoln("Sim UE terminated")
	return nil
}

func ChangeProcedure(ue *simuectx.SimUe) {
	// Check if custom Profile
	if len(ue.ProfileCtx.Iterations) > 0 {
		ue.Log.Infoln("Current UE iteration ", ue.CurrentItr)
		ue.Log.Infoln("CurrentProcIndex  ", ue.CurrentProcIndex)
		itp := ue.ProfileCtx.PIterations[ue.CurrentItr]
		if itp.WaitMap[ue.CurrentProcIndex] != 0 {
			time.Sleep(time.Second * time.Duration(itp.WaitMap[ue.CurrentProcIndex]))
		}
		nextProcIndex := ue.CurrentProcIndex + 1
		nextProcedure, found := itp.ProcMap[nextProcIndex]
		if found == true {
			ue.Log.Infoln("Next Procedure Index, Proc ", nextProcIndex, nextProcedure)
			ue.Procedure = nextProcedure
			ue.CurrentProcIndex = nextProcIndex
			ue.Log.Infoln("Updated procedure to", nextProcedure)
			HandleProcedure(ue)
			return
		}
		if ue.Repeat > 0 {
			ue.Repeat = ue.Repeat - 1
			ue.Log.Infoln("Repeat current iteration : ", itp.Name, ", Repeat Count ", ue.Repeat)
			ue.CurrentProcIndex = 1
			nextProcedure := itp.ProcMap[1]
			ue.Procedure = nextProcedure
			HandleProcedure(ue)
			return
		}

		ue.Log.Infoln("Iteration Complete ", ue.CurrentItr)
		nextItr := itp.NextItr
		if nextItr != "quit" {
			nItr := ue.ProfileCtx.PIterations[nextItr]
			ue.Log.Infoln("Going to next iteration ", nItr.Name)
			ue.CurrentItr = nItr.Name
			ue.CurrentProcIndex = 1
			ue.Repeat = nItr.Repeat
			nextProcedure := nItr.ProcMap[1]
			ue.Procedure = nextProcedure
			HandleProcedure(ue)
			return
		}
		ue.Log.Infoln("QUIT profile ")
	} else {
		// basic predefined profiles
		nextProcedure := ue.ProfileCtx.GetNextProcedure(ue.Procedure)
		if nextProcedure != 0 {
			ue.Procedure = nextProcedure
			ue.Log.Infoln("Updated procedure to", nextProcedure)
			HandleProcedure(ue)
			return
		}
	}
	//no procedure to be executed. Send PROFILE_PASS_EVENT
	SendToProfile(ue, common.PROFILE_PASS_EVENT, nil)
	evt, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, common.PROFILE_PASS_EVENT)
	// This is suppose to be last event..why do we care to get return error ?
	if err != nil {
		ue.Log.Errorln("GetNextEvent failed:", err)
		return
	}
	if evt == common.QUIT_EVENT {
		msg := &common.DefaultMessage{}
		msg.Event = common.QUIT_EVENT
		ue.ReadChan <- msg
	}
	return
}

func HandleProcedure(ue *simuectx.SimUe) {
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
	case common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
		ue.Log.Infoln("Initiating UE Requested PDU Session Release Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.PDU_SESS_REL_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.USER_DATA_PKT_GENERATION_PROCEDURE:
		ue.Log.Infoln("Initiating User Data Packet Generation Procedure")
		msg := &common.UeMessage{}
		msg.UserDataPktCount = ue.ProfileCtx.DataPktCount
		if ue.ProfileCtx.DefaultAs == "" {
			ue.ProfileCtx.DefaultAs = "192.168.250.1" // default destination for AIAB
		}
		msg.DefaultAs = ue.ProfileCtx.DefaultAs
		msg.Event = common.DATA_PKT_GEN_REQUEST_EVENT

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
		msg.Event = common.TRIGGER_AN_RELEASE_EVENT
		SendToGnbUe(ue, msg)
	case common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE:
		ue.Log.Infoln("Initiating UE Triggered Service Request Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.SERVICE_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE:
		ue.Log.Infoln("Waiting for N/W Triggered De-registration Procedure")
	case common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
		ue.Log.Infoln("Waiting for N/W Requested PDU Session Release Procedure")
	}
}
