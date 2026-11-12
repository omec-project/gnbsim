// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	simuectx "github.com/omec-project/gnbsim/simue/context"
	"github.com/omec-project/gnbsim/stats"
)

func HandleProcedureEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.ProfileMessage)
	ue.Procedure = msg.Proc
	// nothing is left over from the previous procedure
	ue.BearerRestorePending = false
	ue.ServiceAccepted = false
	ue.BearersRestored = false
	ue.AcceptExpected = false
	// a request that was not answered must not be resent during another procedure
	stopRetransmission(ue)
	// A Service Accept that arrives late, after the bearers completed the previous service request,
	// still has to be recognised during the procedure that follows, so ServiceCompleted survives.
	// A new service request is the exception: an Accept for that one must not be swallowed.
	if ue.Procedure == common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE {
		ue.ServiceCompleted = false
		ue.IgnoreBearerResult = false
		ue.PassedByAccept = false
	}
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

	// The Initial Context Setup that carried the Registration Accept may also have restored
	// bearers. The procedure is then done only when they are in place as well, as stopping the UE
	// before that would leave the AMF without its Initial Context Setup Response.
	if ue.BearerRestorePending {
		ue.ServiceAccepted = true
		completeRestoration(ue)
		return nil
	}

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
//
// Reaching this at all means the reject matched the UE's outstanding request: the RealUe checks
// that before it reports the message, precisely because the pass below is unconditional and is
// sent before the RealUe could otherwise have looked. The reject is not passed back to the
// RealUe, which has already handled it.
func HandlePduSessModificationRejectEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
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

// HandleBearerSetupPendingEvent notes that the procedure in progress also restores the user plane.
// It then completes only when the service has been accepted and the gNB has reported the bearers,
// whichever comes first, and exactly once.
func HandleBearerSetupPendingEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	// The Service Accept came first and completed the procedure, which is now past undoing. What
	// follows from this restoration is not part of any procedure that is waiting for it.
	// Whichever procedure is running now: the notice is matched to the request by its gNB context.
	if uemsg, ok := msg.(*common.UuMessage); ok && ue.PassedByAccept && uemsg.CtxGeneration == ue.PassedByAcceptId {
		ue.PassedByAccept = false
		ue.IgnoreBearerResult = true
		ue.IgnoreResultId = uemsg.CtxGeneration
		ue.Log.Warnln("bearers are being restored for a service request that is already complete")
		return nil
	}
	// Only a service request and a registration have a NAS half that this waits for. Any other
	// procedure is completed by its own messages, and waiting here would never end.
	nasHalfExpected := false
	switch ue.Procedure {
	case common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE:
		// the gNB says whether the request carries the Service Accept; if it does not, the AMF may
		// send it separately or not at all, and the restored bearers complete the procedure
		if uemsg, ok := msg.(*common.UuMessage); ok {
			nasHalfExpected = len(uemsg.NasPdus) > 0
		}
	case common.REGISTRATION_PROCEDURE:
		// the Registration Complete is always sent, and ends the NAS half
		nasHalfExpected = true
	default:
		return nil
	}
	ue.BearerRestorePending = true
	ue.ServiceAccepted = false // "the NAS half is done": the Service Accept, or the Registration Complete
	ue.BearersRestored = false
	ue.ServiceCompleted = false
	ue.AcceptExpected = nasHalfExpected
	return nil
}

// completeRestoration reports the procedure as done if it is: the bearers are restored and, if the
// request carried the Service Accept, the service has been accepted too.
func completeRestoration(ue *simuectx.SimUe) {
	if !ue.BearersRestored || (ue.AcceptExpected && !ue.ServiceAccepted) {
		return
	}
	// done by the bearers alone: an Accept that comes later is not a second completion
	ue.ServiceCompleted = !ue.ServiceAccepted
	stopRetransmission(ue) // the procedure is done, whether or not an Accept ever came
	ue.BearerRestorePending = false
	ue.ServiceAccepted = false
	ue.BearersRestored = false
	SendProcedureResult(ue)
}

func HandleDataBearerSetupRequestEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	SendToRealUe(ue, msg)
	return nil
}

// failedPduSessions returns the IDs of the sessions that the RealUe reports as not established.
func failedPduSessions(msg common.InterfaceMessage) []int64 {
	uemsg, ok := msg.(*common.UuMessage)
	if !ok {
		return nil
	}
	var failed []int64
	for _, item := range uemsg.DBParams {
		if item != nil && item.PduSess != nil && !item.PduSess.Success {
			failed = append(failed, item.PduSess.PduSessId)
		}
	}
	return failed
}

// HandleDataBearerSetupResponseEvent receives the RealUe's answer to a data bearer setup request.
// It hands the answer to the gNB, which responds to the AMF, but does not complete the procedure
// yet: that depends on the gNB's own result (HandleDataBearerSetupResultEvent).
func HandleDataBearerSetupResponseEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	// evaluated before the message is handed over, as the gNB updates the session entries
	failed := failedPduSessions(msg)

	// the gNB still reports the failed sessions to the AMF (TS 38.413 clause 8.2.1.2)
	SendToGnbUe(ue, msg)

	if len(failed) > 0 {
		// The procedure asked for these sessions, so it did not complete. Returning the error
		// makes HandleEvents report the failure to the profile and tear the UE down, which also
		// stops the sessions that were established.
		return fmt.Errorf("PDU session setup failed for session(s) %v", failed)
	}
	return nil
}

// HandleDataBearerSetupResultEvent completes the procedure once the gNB has answered the AMF and
// has the user plane in place. An error from the gNB fails it, and the event loop tears the UE down.
func HandleDataBearerSetupResultEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	if gnbErr := msg.GetErrorMsg(); gnbErr != nil {
		return fmt.Errorf("gNB failed to set up the data bearers: %w", gnbErr)
	}

	if uemsg, ok := msg.(*common.UuMessage); ok && ue.IgnoreBearerResult && uemsg.CtxGeneration == ue.IgnoreResultId {
		ue.IgnoreBearerResult = false
		ue.Log.Infoln("bearers restored for a service request that was already complete")
		return nil
	}
	if ue.BearerRestorePending {
		// part of a service request: it is done once the service has also been accepted
		ue.BearersRestored = true
		completeRestoration(ue)
		return nil
	}

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
	// returned to HandleEvents, which reports the failure to the profile and tears the UE down
	return fmt.Errorf("data packet generation failed: %v", msg.GetErrorMsg())
}

// retransmitInterval is how long the Service Request waits for an answer before it is resent.
const retransmitInterval = 5 * time.Second

// retransmitMsg resends the request every few seconds, a limited number of times, until stop is
// closed, which is what an answer to this request, the end of its procedure and a quit all do.
func retransmitMsg(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage, count int, interval time.Duration, stop <-chan struct{}) {
	// TBD: Profile should give timeout and number of retransmission as input
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			ue.Log.Debugln("stopped resending the Service Request")
			return
		case <-ue.QuitRequested():
			return
		case <-ticker.C:
			if count == 0 {
				return // nothing left to resend, and no reason to keep waiting
			}
			// An answer, the end of the procedure or a quit may have come while the ticker fired, and
			// then the request is no longer wanted: the select below picks at random between ready
			// cases, so this is looked at first.
			select {
			case <-stop:
				return
			case <-ue.QuitRequested():
				return
			default:
			}
			ue.Log.Debugln("resend Service Request count", count)
			// The send itself gives way to the stop and quit signals, not only to the quit of
			// SendToGnbUe: a worker that stayed blocked on a full queue after its request had been
			// answered could enqueue the old request during the next procedure.
			select {
			case ue.WriteGnbUeChan <- intfcMsg:
			case <-stop:
				return
			case <-ue.QuitRequested():
				return
			}
			count--
		}
	}
}

// startRetransmission resends the request until stopRetransmission, replacing a worker that is
// still running for an earlier one.
func startRetransmission(ue *simuectx.SimUe, intfcMsg common.InterfaceMessage, count int) {
	stopRetransmission(ue)
	stop := make(chan struct{})
	ue.RetransmitStop = stop
	ue.RetransmitWg.Go(func() { retransmitMsg(ue, intfcMsg, count, retransmitInterval, stop) })
}

// stopRetransmission stops the worker that resends the request, if there is one, and waits until
// it has returned. Its send can still win against the closed stop channel when the queue has room,
// so without the wait a request could be enqueued after the event loop had moved on. The worker's
// blocking points all give way to stop, so the wait is short.
func stopRetransmission(ue *simuectx.SimUe) {
	if ue.RetransmitStop != nil {
		close(ue.RetransmitStop)
		ue.RetransmitStop = nil
	}
	ue.RetransmitWg.Wait()
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
		startRetransmission(ue, intfcMsg, 2)
	}

	ue.Log.Debugln("sent Service Request Event to the network")
	return nil
}

func HandleServiceAcceptEvent(ue *simuectx.SimUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	// An Accept for a service request that the bearers already completed is neither an answer to a
	// request that is in progress nor an event of the procedure that is now running, so it is
	// consumed before either is looked at.
	if ue.ServiceCompleted {
		ue.ServiceCompleted = false
		ue.Log.Infoln("service accepted after the bearers had already completed the procedure")
		return nil
	}
	stopRetransmission(ue) // the answer to this request has come
	err = ue.ProfileCtx.CheckCurrentEvent(ue.Procedure, common.SERVICE_REQUEST_EVENT,
		intfcMsg.GetEventType())
	if err != nil {
		ue.Log.Errorln("CheckCurrentEvent returned:", err)
		return err
	}

	msg := intfcMsg.(*common.UeMessage)
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.SVC_ACCEPT_IN, Id: msg.Id}
	stats.LogStats(e)

	// Service Accept completes the UE-triggered service request procedure, unless the gNB is still
	// restoring the user plane: that is part of the procedure, and stopping the UE before it is
	// done would leave the AMF without its Initial Context Setup Response.
	if ue.BearerRestorePending {
		ue.ServiceAccepted = true
		completeRestoration(ue)
		return nil
	}
	// the gNB has not said anything about bearers, so the Accept completes it on its own
	ue.PassedByAccept = true
	ue.PassedByAcceptId = 0
	if ctxs := ue.GnbContexts(); len(ctxs) > 0 {
		// the context created for this service request is the last one that was recorded
		ue.PassedByAcceptId = ctxs[len(ctxs)-1].Generation
	}
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

// gnbTeardownWarnInterval is how often a SimUe that waits for a gNB context to terminate says so.
const gnbTeardownWarnInterval = 30 * time.Second

func waitForGnbContext(ue *simuectx.SimUe, c *gnbctx.GnbCpUe) {
	ticker := time.NewTicker(gnbTeardownWarnInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.Done:
			return
		case <-ticker.C:
			ue.Log.Warnln("still waiting for the gNB UE context to terminate")
		}
	}
}

// The RealUe sends to ReadChan without being able to give up, and this goroutine is the only one
// that reads it. While the SimUe quits it must keep reading, and discard what it reads, or a
// RealUe that is blocked on a full queue can neither receive its quit nor terminate.

func sendToRealUeWhileDraining(ue *simuectx.SimUe, msg common.InterfaceMessage) {
	for {
		select {
		case ue.WriteRealUeChan <- msg:
			return
		case dropped := <-ue.ReadChan:
			ue.Log.Debugln("discarding", dropped.GetEventType(), "while quitting")
		}
	}
}

func waitForRealUe(ue *simuectx.SimUe) {
	done := make(chan struct{})
	go func() {
		ue.WaitGrp.Wait()
		close(done)
	}()
	for {
		select {
		case <-done:
			return
		case dropped := <-ue.ReadChan:
			ue.Log.Debugln("discarding", dropped.GetEventType(), "while quitting")
		}
	}
}

func HandleQuitEvent(ue *simuectx.SimUe,
	msg common.InterfaceMessage,
) (err error) {
	// Every gNB context created for the UE is asked directly, which cannot be dropped, and which
	// covers a target that the UE has not moved to and a source that it has left. Each handles the
	// events already queued for it first, then terminates; one that already quit ignores this.
	stopRetransmission(ue)
	gnbCtxs := ue.GnbContexts()
	for _, c := range gnbCtxs {
		c.RequestQuit()
	}
	// Also queue a quit for the current gNB context, but never block on it: that queue may be full
	// because its owner is waiting for this SimUe, which has stopped reading.
	if ue.WriteGnbUeChan != nil {
		select {
		case ue.WriteGnbUeChan <- msg:
		default:
			ue.Log.Warnln("the gNB UE queue is full, relying on the quit request")
		}
	}
	// Guarded like the channel above it, and for the reason this function itself creates: the
	// line below nils the channel, so a second quit for the same UE would send on a nil channel
	// and block its goroutine for good. Nothing reaches here twice today; the guard costs a line
	// and the deadlock costs a hung worker with no log to say why.
	if ue.WriteRealUeChan != nil {
		sendToRealUeWhileDraining(ue, msg)
	}
	ue.WriteRealUeChan = nil
	waitForRealUe(ue)

	// The SUPI is freed when this SimUe's event loop ends, so wait for the gNB side as well. There
	// is no timeout: giving up would free the SUPI while gNB workers are still live.
	for _, c := range gnbCtxs {
		waitForGnbContext(ue, c)
	}
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
	case common.NW_PDU_SESSION_MODIFICATION_PROCEDURE:
		ue.Log.Infoln("Waiting for N/W Requested PDU Session Modification Procedure")
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
		//
		// It fails the procedure rather than only logging, because logging alone leaves the
		// profile waiting out perUserTimeout and reporting "profile timeout": the same silence
		// this branch exists to end, arriving a minute later with the wrong explanation.
		//
		// The failure goes through HandleErrorEvent rather than straight to the profile, because
		// reporting the result is only half of ending a procedure: the other half is the QUIT that
		// stops this UE's workers and releases its bearer at the gNB. Reported without it, the
		// profile moves on while the SimUe, the RealUe and the gNB context stay up.
		err := fmt.Errorf("no handler for procedure %v: it will not start, and nothing will be sent",
			ue.Procedure)
		ue.Log.Errorln(err)
		errMsg := &common.UeMessage{}
		errMsg.Event = common.ERROR_EVENT
		errMsg.Error = err
		if handleErr := HandleErrorEvent(ue, errMsg); handleErr != nil {
			ue.Log.Errorln("failed to handle the error event:", handleErr)
		}
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
