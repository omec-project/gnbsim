// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	simuectx "github.com/omec-project/gnbsim/simue/context"
	"github.com/omec-project/gnbsim/stats"
)

func HandleProcedureEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.ProfileMessage)
	ue.Procedure = msg.Proc
	ue.Log.Infoln("Start new procedure ", ue.Procedure)
	HandleProcedure(ue)
	return nil
}

func HandleRegRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	SendToGnbUe(ue, intfcMsg)
	return nil
}

func HandleRegRejectEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.REG_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	return nil
}

func HandleAuthRequestEvent(ue *simuectx.SimUe,
	intfMsg common.InterfaceMessage,
) (err error) {
	msg := intfMsg.(*common.UeMessage)
	// checking as per profile if Authentication Request Message is expected
	// from 5G Core against Registration Request message sent by RealUE
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.REG_REQUEST_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.AUTH_REQ_IN, Id: msg.Id}
	stats.LogStats(e)

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
	intfcMsg common.InterfaceMessage,
) (err error) {
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
	intfcMsg common.InterfaceMessage,
) (err error) {
	// TODO: Should check if SecModCommandEvent event is expected

	msg := intfcMsg.(*common.UeMessage)
	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}

	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.SECM_CMD_IN, Id: msg.Id}
	stats.LogStats(e)

	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

func HandleSecModCompleteEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
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
	intfcMsg common.InterfaceMessage,
) (err error) {
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
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.REG_ACCEPT_EVENT, msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Registration Complete to the network")

	// Current Procedure is complete. Move to next one
	SendProcedureResult(ue)
	return nil
}

func HandleDeregRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Traceln("Sent Deregistration Request to the network")

	return nil
}

func HandleDeregAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.DEREG_ACC_IN, Id: msg.Id}
	stats.LogStats(e)

	return nil
}

func HandlePduSessEstRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

func HandlePduSessEstAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
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
	intfcMsg common.InterfaceMessage,
) (err error) {
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.PDU_SESS_EST_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	return nil
}

func HandlePduSessReleaseRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

func HandlePduSessReleaseCommandEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
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
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

func HandleDlInfoTransferEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	SendToRealUe(ue, msg)
	return nil
}

func HandleDataBearerSetupRequestEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	SendToRealUe(ue, msg)
	return nil
}

func HandleDataBearerSetupResponseEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	SendToGnbUe(ue, msg)

	// Current Procedure is complete. Move to next one
	SendProcedureResult(ue)
	return nil
}

func HandleDataBearerReleaseRequestEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	// This event is sent by gNB component after it has sent
	// PDU Session Resource Release Complete over N2, However the PDU Sesson
	// routines in the RealUE will be terminated while processing PDU Session
	// Release Complete which will also release the communication links
	// (go channels) with the gNB
	// Current Procedure is complete. Move to next one
	SendProcedureResult(ue)
	return nil
}

func HandleDataPktGenSuccessEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	// Current Procedure is complete. Move to next one
	SendProcedureResult(ue)
	return nil
}

func HandleDataPktGenFailureEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	ue.Log.Traceln("HandleDataPktGenFailureEvent")
	SendToProfile(ue, common.PROC_FAIL_EVENT, msg.GetErrorMsg())
	return nil
}

func retransmitMsg(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage, count int) {
	// TBD: Profile should give timeout and number of retransmission as input
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ue.MsgRspReceived:
			ue.Log.Traceln("Received Service Accept Message ")
			ticker.Stop()
			return
		case <-ticker.C:
			if count > 0 {
				ue.Log.Traceln("Resend Service Request count ", count)
				SendToGnbUe(ue, intfcMsg)
				count = count - 1
			}
		}
	}
}

func HandleServiceRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	err = ConnectToGnb(ue)
	if err != nil {
		return fmt.Errorf("failed to connect gnb %v", err)
	}

	SendToGnbUe(ue, intfcMsg)
	if ue.ProfileCtx.RetransMsg {
		go retransmitMsg(ue, intfcMsg, 2)
	}

	ue.Log.Traceln("Sent Service Request Event to the network")
	return nil
}

func HandleServiceAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	if ue.ProfileCtx.RetransMsg {
		ue.MsgRspReceived <- true // feedback loop
	}
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.SERVICE_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg := intfcMsg.(*common.UeMessage)
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.SVC_ACCEPT_IN, Id: msg.Id}
	stats.LogStats(e)

	return nil
}

func HandleConnectionReleaseRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
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
		/*
			msg := &common.UeMessage{}
			msg.Event = common.QUIT_EVENT
			ue.ReadChan <- msg
		*/
		// Nothing else to execute. Tell profile we are done.
		ue.Log.Traceln("debug2")
		SendToProfile(ue, common.PROC_PASS_EVENT, nil)
		return nil
	}
	SendToRealUe(ue, msg)
	// Current Procedure is complete. Move to next one
	SendProcedureResult(ue)

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
	intfcMsg common.InterfaceMessage,
) (err error) {
	ue.Log.Traceln("debug3")
	SendToProfile(ue, common.PROC_FAIL_EVENT, intfcMsg.GetErrorMsg())

	msg := &common.UuMessage{}
	msg.Event = common.QUIT_EVENT
	err = HandleQuitEvent(ue, msg)
	if err != nil {
		ue.Log.Warnln("failed to handle quiet event", err)
	}
	return nil
}

func HandleQuitEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	if ue.WriteGnbUeChan != nil {
		SendToGnbUe(ue, msg)
	}
	SendToRealUe(ue, msg)
	ue.WriteRealUeChan = nil
	ue.WaitGrp.Wait()
	ue.Log.Infoln("Sim UE terminated")
	return nil
}

// TODO : accept result, 1. pass or 2. Fail (with error)
func SendProcedureResult(ue *simuectx.SimUe) {
	ue.Log.Traceln("Sending Procedure Result to Profile : PASS")
	SendToProfile(ue, common.PROC_PASS_EVENT, nil)
	// e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_PROC_END, Id: 0}
	// stats.LogStats(e)
}

func HandleProcedure(ue *simuectx.SimUe) {
	switch ue.Procedure {
	case common.REGISTRATION_PROCEDURE:
		ue.Log.Infoln("Initiating Registration Procedure")
		e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_PROC_START, Id: 0}
		stats.LogStats(e)
		msg := &common.UeMessage{}
		msg.Event = common.REG_REQUEST_EVENT
		msg.Id = 0
		SendToRealUe(ue, msg)
	case common.PDU_SESSION_ESTABLISHMENT_PROCEDURE:
		ue.Log.Infoln("Initiating UE Requested PDU Session Establishment Procedure")
		e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_PROC_START, Id: 0}
		stats.LogStats(e)
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
		msg.UserDataPktInterval = ue.ProfileCtx.DataPktInt
		msg.DefaultAs = ue.ProfileCtx.DefaultAs
		msg.Event = common.DATA_PKT_GEN_REQUEST_EVENT

		/* TODO: Solve timing issue. Currently UE may start sending user data
		 * before gnb has successfully sent PDU Session Resource Setup Response
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
