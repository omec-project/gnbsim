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
	ue.Log.Infoln("start new procedure:", ue.Procedure)
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
	ue.Log.Infoln("next event:", nextEvent)
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
	ue.Log.Debugln("sending Authentication Response to the network")
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
	ue.Log.Debugln("handling Security Mode Complete Event")

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.SEC_MOD_COMMAND_EVENT,
		msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Debugln("sent Security Mode Complete to the network")
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
	ue.Log.Debugln("sent Registration Complete to the network")

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
	ue.Log.Debugln("sent Deregistration Request to the network")

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

func HandlePduSessEstRequestEvent(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage) (err error) {
	// Safe type assertions
	if intfcMsg == nil {
		err := fmt.Errorf("HandlePduSessEstRequestEvent: intfcMsg is nil")
		ue.Log.Errorln(err)
		return err
	}

	msg, ok := intfcMsg.(*common.UuMessage)
	if !ok {
		err := fmt.Errorf("HandlePduSessEstRequestEvent: expected *common.UuMessage, got %T", intfcMsg)
		ue.Log.Errorln(err)
		return err
	}
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
	ue.Log.Infoln("next event:", nextEvent)
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
	ue.Log.Infoln("next event:", nextEvent)
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

// HandlePduSessModificationRequestEvent forwards the UE's built request to the gNB.
//
// The procedure is started by HandleProcedure sending this event straight to the RealUe, so what
// arrives here is the RealUe's answer coming back — a UuMessage carrying the encoded NAS, which
// becomes an uplink transfer.
func HandlePduSessModificationRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	return nil
}

// HandlePduSessModificationRejectEvent takes the network's refusal.
//
// A refusal is the expected outcome, not a failure: the core declines every UE-requested
// modification. The procedure therefore passes when the reject arrives, and would fail by timing
// out if the network said nothing at all — which is what it did before it was taught to refuse.
func HandlePduSessModificationRejectEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	SendToRealUe(ue, msg)

	SendProcedureResult(ue)
	return nil
}

// HandlePduSessModificationCommandEvent takes a modification the network started on its own.
//
// Unlike the release command there is no UE-side procedure to reconcile against: the command
// arrives unprompted with no procedure transaction identity, so the arrival is what begins the
// procedure. The UE is switched onto it before the answer is decided, or the profile's event map
// would be consulted for whatever procedure happened to be running.
func HandlePduSessModificationCommandEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)

	if ue.Procedure != common.NW_PDU_SESSION_MODIFICATION_PROCEDURE {
		ue.Log.Infoln("network started a PDU session modification; switching to it from",
			ue.Procedure)
		ue.Procedure = common.NW_PDU_SESSION_MODIFICATION_PROCEDURE
	}

	if ue.ProfileCtx.WithholdModificationComplete {
		// Deliberately silent. The network should retransmit and then abandon, and the UE keeps
		// the parameters it already had.
		ue.Log.Infoln("withholding the modification complete, as the profile asks")
		return nil
	}

	nextEvent, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, msg.Event)
	if err != nil {
		ue.Log.Errorln("GetNextEvent returned:", err)
		return err
	}
	ue.Log.Infoln("next event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)
	return nil
}

// HandlePduSessModificationCompleteEvent forwards the UE's acknowledgement to the gNB, and treats
// the procedure as finished once it is on its way.
func HandlePduSessModificationCompleteEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)

	SendProcedureResult(ue)
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
	ue.Log.Debugln("handleDataPktGenFailureEvent")
	SendToProfile(ue, common.PROC_FAIL_EVENT, msg.GetErrorMsg())
	return nil
}

func retransmitMsg(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage, count int) {
	// TBD: Profile should give timeout and number of retransmission as input
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ue.MsgRspReceived:
			ue.Log.Debugln("received Service Accept Message")
			ticker.Stop()
			return
		case <-ticker.C:
			if count > 0 {
				ue.Log.Debugln("resend Service Request count", count)
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

	ue.Log.Debugln("sent Service Request Event to the network")
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

	// Service Accept completes the UE-triggered service request procedure.
	SendProcedureResult(ue)

	return nil
}

func HandleConnectionReleaseRequestEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)

	// After a successful N2 handover the source gNB sends this event when it
	// receives UEContextReleaseCommand from the AMF. At that point WriteGnbUeChan
	// already points to the target gNB, so we must NOT nil it out or complete the
	// current procedure (which may be USER_DATA_PKT_GENERATION on the target gNB).
	if msg.TriggeringEvent == common.HO_COMMAND_EVENT {
		ue.Log.Infoln("ignoring CONNECTION_RELEASE_REQUEST from source gNB after N2 handover")
		return nil
	}

	if ue.Procedure == common.AN_RELEASE_PROCEDURE {
		err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.TRIGGER_AN_RELEASE_EVENT,
			common.CONNECTION_RELEASE_REQUEST_EVENT)
		if err != nil {
			return err
		}
	}

	ue.WriteGnbUeChan = nil

	if msg.TriggeringEvent == common.DEREG_REQUEST_UE_ORIG_EVENT {
		// Nothing else to execute. Tell profile we are done.
		ue.Log.Debugln("debug2")
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
	ue.Log.Infoln("next event:", nextEvent)
	msg.Event = nextEvent
	SendToRealUe(ue, msg)

	return nil
}

func HandleNwDeregAcceptEvent(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage) (err error) {
	ue.Log.Debugln("handling Dereg Accept Event")

	msg := intfcMsg.(*common.UuMessage)
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.DEREG_REQUEST_UE_TERM_EVENT,
		msg.Event)
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg.Event = common.UL_INFO_TRANSFER_EVENT
	SendToGnbUe(ue, msg)
	ue.Log.Debugln("sent Dereg Accept to the network")
	return nil
}

func HandleErrorEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	ue.Log.Debugln("debug3")
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
	ue.Log.Debugln("sending Procedure Result to Profile : PASS")
	SendToProfile(ue, common.PROC_PASS_EVENT, nil)
	// e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_PROC_END, Id: 0}
	// stats.LogStats(e)
}

func HandleProcedure(ue *simuectx.SimUe) {
	switch ue.Procedure {
	case common.REGISTRATION_PROCEDURE:
		ue.Log.Infoln("initiating Registration Procedure")
		e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_PROC_START, Id: 0}
		stats.LogStats(e)
		msg := &common.UeMessage{}
		msg.Event = common.REG_REQUEST_EVENT
		msg.Id = 0
		SendToRealUe(ue, msg)
	case common.PDU_SESSION_ESTABLISHMENT_PROCEDURE:
		ue.Log.Infoln("initiating UE Requested PDU Session Establishment Procedure")
		e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_PROC_START, Id: 0}
		stats.LogStats(e)
		msg := &common.UeMessage{}
		msg.Event = common.PDU_SESS_EST_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.UE_REQUESTED_PDU_SESSION_MODIFICATION_PROCEDURE:
		ue.Log.Infoln("initiating UE Requested PDU Session Modification Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.PDU_SESS_MOD_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
		ue.Log.Infoln("initiating UE Requested PDU Session Release Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.PDU_SESS_REL_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.USER_DATA_PKT_GENERATION_PROCEDURE:
		ue.Log.Infoln("initiating User Data Packet Generation Procedure")
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
		ue.Log.Infoln("initiating UE Initiated Deregistration Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.DEREG_REQUEST_UE_ORIG_EVENT
		SendToRealUe(ue, msg)
	case common.AN_RELEASE_PROCEDURE:
		ue.Log.Infoln("initiating AN Release Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.TRIGGER_AN_RELEASE_EVENT
		SendToGnbUe(ue, msg)
	case common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE:
		ue.Log.Infoln("initiating UE Triggered Service Request Procedure")
		msg := &common.UeMessage{}
		msg.Event = common.SERVICE_REQUEST_EVENT
		SendToRealUe(ue, msg)
	case common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE:
		ue.Log.Infoln("Waiting for N/W Triggered De-registration Procedure")
	case common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
		ue.Log.Infoln("Waiting for N/W Requested PDU Session Release Procedure")
	case common.N2_HANDOVER_PROCEDURE:
		ue.Log.Infoln("initiating N2 Handover Procedure")
		// Pre-register with target gNB so gnbamfworker can route HandoverRequest
		if err := ConnectToTargetGnb(ue); err != nil {
			ue.Log.Errorln("ConnectToTargetGnb failed:", err)
			SendToProfile(ue, common.PROC_FAIL_EVENT, err)
			return
		}
		// Tell source gNB to send HandoverRequired to AMF
		msg := &common.UuMessage{}
		msg.Event = common.TRIGGER_HO_EVENT
		msg.TargetGnbName = ue.ProfileCtx.TargetGnbName
		SendToGnbUe(ue, msg)
	default:
		// A procedure registered in procedures.go with no case here starts, logs that it started,
		// and then does nothing — the UE never sends anything and the silence looks like the
		// network failing to answer rather than the UE failing to ask. That cost real time once.
		ue.Log.Errorf("no handler for procedure %v: it will not start, and nothing will be sent",
			ue.Procedure)
	}
}

// HandleHandoverCommandEvent is called when SimUe receives HO_COMMAND_EVENT from
// the source gNB. It switches the active gNB to the target gNB and sends
// HO_NOTIFY_EVENT via the target gNB, which will trigger the data bearer
// re-setup with the RealUE. The N2_HANDOVER_PROCEDURE completes once
// DATA_BEARER_SETUP_RESPONSE_EVENT is received (via HandleDataBearerSetupResponseEvent).
func HandleHandoverCommandEvent(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage) (err error) {
	ue.Log.Infoln("received HandoverCommand - switching to target gNB")

	if ue.TargetWriteGnbUeChan == nil {
		return fmt.Errorf("target gNB channel is nil, was ConnectToTargetGnb called?")
	}

	// Switch SimUe to communicate through the target gNB
	ue.WriteGnbUeChan = ue.TargetWriteGnbUeChan
	ue.GnB = ue.TargetGnB
	ue.TargetWriteGnbUeChan = nil
	ue.TargetGnB = nil

	ue.Log.Infoln("switched to target gNB:", ue.GnB.GnbName)

	// Tell target gNB to re-setup data bearers with RealUE and then send
	// HandoverNotify. The procedure result is deferred until the
	// DATA_BEARER_SETUP_RESPONSE arrives back from the target gNB.
	notifyMsg := &common.UuMessage{}
	notifyMsg.Event = common.HO_NOTIFY_EVENT
	SendToGnbUe(ue, notifyMsg)
	return nil
}
