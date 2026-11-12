// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	"github.com/omec-project/gnbsim/gnodeb"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	profctx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/gnbsim/profile/util"
	"github.com/omec-project/gnbsim/realue"
	simuectx "github.com/omec-project/gnbsim/simue/context"
)

// InitUE creates a SimUe and starts it. A connection failure is returned rather than sent to the
// profile: the profile only reads its result channel once it executes, which is after
// initialisation, so sending there would block the caller forever.
func InitUE(imsiStr string, gnb *gnbctx.GNodeB, profile *profctx.Profile, result chan *common.ProfileMessage) (*simuectx.SimUe, error) {
	simUe := simuectx.NewSimUe(imsiStr, gnb, profile, result)
	if err := Init(simUe); err != nil { // Initialize simUE, realUE & wait for events
		simuectx.DeleteSimUe(simUe)
		return nil, err
	}
	return simUe, nil
}

func Init(simUe *simuectx.SimUe) error {
	err := ConnectToGnb(simUe)
	if err != nil {
		err = fmt.Errorf("failed to connect to gnodeb: %v", err)
		simUe.Log.Errorln(err)
		return err
	}

	simUe.WaitGrp.Add(1)
	go func() {
		defer simUe.WaitGrp.Done()
		realue.Init(simUe.RealUe)
	}()

	go HandleEvents(simUe)
	simUe.Log.Infoln("SIM UE Init complete")
	return nil
}

func ConnectToGnb(simUe *simuectx.SimUe) error {
	uemsg := common.UuMessage{}
	uemsg.Event = common.CONNECTION_REQUEST_EVENT
	uemsg.CommChan = simUe.ReadChan
	uemsg.Supi = simUe.Supi

	var err error
	var gnbCtx *gnbctx.GnbCpUe
	gNb := simUe.GnB
	simUe.WriteGnbUeChan, gnbCtx, err = gnodeb.RequestConnection(gNb, &uemsg)
	if err != nil {
		simUe.Log.Errorf("connecting to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
			gNb.GnbN2Ip, gNb.GnbN2Port)
		return err
	}

	simUe.AddGnbContext(gnbCtx)
	simUe.Log.Infof("connected to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
		gNb.GnbN2Ip, gNb.GnbN2Port)
	return nil
}

// ConnectToTargetGnb pre-registers the UE with the target gNB before N2 handover.
// It allocates a GnbCpUe on the target gNB and stores the channel for later use.
// Must be called after the source gNB has obtained an AMF UE NGAP ID (i.e., after ICS).
func ConnectToTargetGnb(simUe *simuectx.SimUe) error {
	if simUe.ProfileCtx.TargetGnbName == "" {
		return fmt.Errorf("targetGnbName not set in profile")
	}

	// Find the source GnbCpUe (keyed by simUe.ReadChan as the WriteUeChan)
	sourceGnbCpUe := simUe.GnB.GnbUes.GetGnbCpUeByUeChan(simUe.ReadChan)
	if sourceGnbCpUe == nil {
		return fmt.Errorf("source GnbCpUe not found for SUPI: %s", simUe.Supi)
	}

	targetGnb, err := factory.AppConfig.Configuration.GetGNodeB(simUe.ProfileCtx.TargetGnbName)
	if err != nil {
		return fmt.Errorf("failed to get target gNB: %w", err)
	}
	simUe.TargetGnB = targetGnb

	targetChan, targetCtx, err := gnodeb.RegisterHandoverTarget(targetGnb, sourceGnbCpUe.AmfUeNgapId, simUe.ReadChan)
	if err != nil {
		return fmt.Errorf("failed to register target gNB: %w", err)
	}
	simUe.AddGnbContext(targetCtx)
	simUe.TargetWriteGnbUeChan = targetChan

	simUe.Log.Infof("pre-registered with target gNB: %v (AMF-UE-NGAP-ID: %v)",
		targetGnb.GnbName, sourceGnbCpUe.AmfUeNgapId)
	return nil
}

// nextEvent returns the next event for the SimUe. A quit request is returned as a QUIT_EVENT and
// takes precedence over queued events.
func nextEvent(ue *simuectx.SimUe) (common.InterfaceMessage, bool) {
	quit := ue.QuitRequested()
	quitMsg := &common.DefaultMessage{Event: common.QUIT_EVENT}
	select {
	case <-quit:
		return quitMsg, true
	default:
	}
	select {
	case msg, ok := <-ue.ReadChan:
		return msg, ok
	case <-quit:
		return quitMsg, true
	}
}

func HandleEvents(ue *simuectx.SimUe) {
	// Once the SimUe has terminated nothing looks it up any more and its SUPI can be used by
	// another profile. The reservation is released here, and not when the profile stops waiting
	// for the UE, so that two UEs with the same SUPI never have live workers at the same time.
	defer ue.MarkTerminated() // runs last, after the SUPI is free
	defer simuectx.ReleaseSupi(ue.Supi)
	defer simuectx.DeleteSimUe(ue)

	var err error
	for {
		msg, ok := nextEvent(ue)
		if !ok {
			return
		}
		event := msg.GetEventType()
		ue.Log.Infoln("handling event:", event)

		switch event {
		case common.PROC_START_EVENT:
			err = HandleProcedureEvent(ue, msg)
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
		case common.PDU_SESS_REL_REQUEST_EVENT:
			err = HandlePduSessReleaseRequestEvent(ue, msg)
		case common.PDU_SESS_REL_COMMAND_EVENT:
			err = HandlePduSessReleaseCommandEvent(ue, msg)
		case common.PDU_SESS_MOD_REQUEST_EVENT:
			err = HandlePduSessModificationRequestEvent(ue, msg)
		case common.PDU_SESS_MOD_REJECT_EVENT:
			err = HandlePduSessModificationRejectEvent(ue, msg)
		case common.PDU_SESS_MOD_COMMAND_EVENT:
			err = HandlePduSessModificationCommandEvent(ue, msg)
		case common.PDU_SESS_MOD_COMPLETE_EVENT:
			err = HandlePduSessModificationCompleteEvent(ue, msg)
		case common.PDU_SESS_EST_ACCEPT_EVENT:
			err = HandlePduSessEstAcceptEvent(ue, msg)
		case common.PDU_SESS_EST_REJECT_EVENT:
			err = HandlePduSessEstRejectEvent(ue, msg)
		case common.PDU_SESS_REL_COMPLETE_EVENT:
			err = HandlePduSessReleaseCompleteEvent(ue, msg)
		case common.DL_INFO_TRANSFER_EVENT:
			err = HandleDlInfoTransferEvent(ue, msg)
		case common.DATA_BEARER_SETUP_REQUEST_EVENT:
			err = HandleDataBearerSetupRequestEvent(ue, msg)
		case common.DATA_BEARER_SETUP_RESPONSE_EVENT:
			err = HandleDataBearerSetupResponseEvent(ue, msg)
		case common.BEARER_SETUP_PENDING_EVENT:
			err = HandleBearerSetupPendingEvent(ue, msg)
		case common.DATA_BEARER_SETUP_RESULT_EVENT:
			err = HandleDataBearerSetupResultEvent(ue, msg)
		case common.DATA_BEARER_RELEASE_REQUEST_EVENT:
			err = HandleDataBearerReleaseRequestEvent(ue, msg)
		case common.DATA_PKT_GEN_SUCCESS_EVENT:
			err = HandleDataPktGenSuccessEvent(ue, msg)
		case common.DATA_PKT_GEN_FAILURE_EVENT:
			err = HandleDataPktGenFailureEvent(ue, msg)
		case common.SERVICE_REQUEST_EVENT:
			err = HandleServiceRequestEvent(ue, msg)
		case common.SERVICE_ACCEPT_EVENT:
			err = HandleServiceAcceptEvent(ue, msg)
		case common.CONNECTION_RELEASE_REQUEST_EVENT:
			err = HandleConnectionReleaseRequestEvent(ue, msg)
		case common.DEREG_REQUEST_UE_TERM_EVENT:
			err = HandleNwDeregRequestEvent(ue, msg)
		case common.DEREG_ACCEPT_UE_TERM_EVENT:
			err = HandleNwDeregAcceptEvent(ue, msg)
		case common.HO_COMMAND_EVENT:
			err = HandleHandoverCommandEvent(ue, msg)
		case common.ERROR_EVENT:
			ue.Log.Warnln("event:", event, " received error")
			err = HandleErrorEvent(ue, msg)
			if err != nil {
				ue.Log.Warnln("failed to handle error event:", err)
			}
			return
		case common.QUIT_EVENT:
			err = HandleQuitEvent(ue, msg)
			if err != nil {
				ue.Log.Warnln("failed to handle quiet event:", err)
			}
			return
		default:
			ue.Log.Warnln("event:", event, "is not supported")
		}

		if err != nil {
			ue.Log.Errorln("failed to handle event:", event, "Error:", err)
			msg := &common.UeMessage{}
			msg.Error = err
			msg.Event = common.ERROR_EVENT
			err = HandleErrorEvent(ue, msg)
			if err != nil {
				ue.Log.Errorln("failed to handle error event:", err)
			}
			return
		}
	}
}

func SendToRealUe(ue *simuectx.SimUe, msg common.InterfaceMessage) {
	ue.Log.Debugln("sending", msg.GetEventType(), "to RealUe")
	// A full queue must not keep the event loop from reaching its teardown, which is what a
	// quit request asks for: the message is no longer wanted by then.
	select {
	case ue.WriteRealUeChan <- msg:
	case <-ue.QuitRequested():
		ue.Log.Debugln("dropped", msg.GetEventType(), "for the RealUe, the SimUe is quitting")
	}
}

func SendToGnbUe(ue *simuectx.SimUe, msg common.InterfaceMessage) {
	ue.Log.Debugln("sending", msg.GetEventType(), "to GnbUe")
	select {
	case ue.WriteGnbUeChan <- msg:
	case <-ue.QuitRequested():
		ue.Log.Debugln("dropped", msg.GetEventType(), "for the GnbUe, the SimUe is quitting")
	}
}

func SendToProfile(ue *simuectx.SimUe, event common.EventType, errMsg error) {
	ue.Log.Debugln("sending", event, "to Profile routine")
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Supi = ue.Supi
	msg.Proc = ue.Procedure
	msg.Error = errMsg
	// The profile stops reading once it has given up on this UE (timeout). Without the quit case
	// the event loop would block here forever and could never run the teardown.
	select {
	case ue.WriteProfileChan <- msg:
		ue.Log.Debugln("sent ", event, "to Profile routine")
	case <-ue.QuitRequested():
		ue.Log.Debugln("dropped", event, "for Profile routine, SimUe is quitting")
	}
}

func RunProcedure(simUe *simuectx.SimUe, procedure common.ProcedureType) {
	util.SendToSimUe(simUe, common.PROC_START_EVENT, procedure)
}

// QuitUe starts the teardown of the SimUe, which stops its RealUe and GnbUe workers, releases the
// gNB UE NGAP ID and, once done, frees the SUPI. It returns without waiting and can be called
// more than once.
func QuitUe(pCtx *profctx.ProfileUeContext) {
	if pCtx.Quit != nil {
		pCtx.Quit()
	}
}

// teardownWarnInterval is how often a caller that waits for a UE to terminate says that it is
// still waiting.
const teardownWarnInterval = 30 * time.Second

// QuitUeAndWait starts the teardown of the SimUe and waits until it has finished and the SUPI is
// free. The profile runners start the next profile as soon as a UE's execution returns, and that
// profile may use the same IMSI. The wait is unbounded: giving up would let a second UE with the
// same SUPI run next to workers that are still live, so a stuck teardown is reported instead.
func QuitUeAndWait(pCtx *profctx.ProfileUeContext) {
	QuitUe(pCtx)
	if pCtx.Terminated == nil {
		return
	}
	ticker := time.NewTicker(teardownWarnInterval)
	defer ticker.Stop()
	for {
		select {
		case <-pCtx.Terminated:
			return
		case <-ticker.C:
			pCtx.Log.Warnln("still waiting for the UE to terminate")
		}
	}
}

func ImsiStateMachine(profile *profctx.Profile, pCtx *profctx.ProfileUeContext, imsiStr string, summaryChan chan common.InterfaceMessage) error {
	var no_more_proc bool
	var proc_fail bool

	// whichever way this returns, the UE must be gone, and its SUPI free, before the caller moves on
	defer QuitUeAndWait(pCtx)

	procedure, err := profile.GetNextProcedure(pCtx, 0)
	if err != nil {
		QuitUe(pCtx)
		return fmt.Errorf("imsi:%v, %v", imsiStr, err)
	}
	if procedure == common.UNKNOWN_PROCEDURE {
		QuitUe(pCtx)
		return fmt.Errorf("imsi:%v, no procedure to execute for profile %v", imsiStr, profile.Name)
	}
	for {
		// select procedure to execute for imsi
		simUe := simuectx.GetSimUe(imsiStr)
		if simUe == nil {
			return fmt.Errorf("imsi:%v, simue context not found", imsiStr)
		}
		pCtx.Log.Infoln("execute procedure ", procedure)
		// proc result -  success, fail or timeout
		timeout := time.Duration(profile.PerUserTimeout) * time.Second
		ticker := time.NewTicker(timeout)
		start_ts := time.Now()
		// Ask simUe to just run procedure and return result
		go RunProcedure(simUe, procedure)
		pCtx.Log.Infoln("Waiting for procedure result from imsiStateMachine")
		select {
		case <-ticker.C:
			err = fmt.Errorf("imsi:%v, profile timeout", imsiStr)
			pCtx.Log.Infoln("Procedure Result: FAIL,", err)
			// do not leave the UE's workers, and a procedure still in flight, running
			QuitUe(pCtx)
			proc_fail = true
		case msg := <-pCtx.ReadChan:
			pCtx.Log.Infoln("imsiStateMachine received result")
			end_ts := time.Now()
			diff := end_ts.Sub(start_ts)
			pCtx.Log.Infof("procedure: %v, status: %v, E2E latency [ms]: %v", procedure.String(), msg.GetEventType().String(), diff.Milliseconds())

			switch msg.Event {
			case common.PROC_PASS_EVENT:
				pCtx.Log.Infoln("Procedure Result: PASS, imsi:", msg.Supi)
				procedure, err = profile.GetNextProcedure(pCtx, simUe.Procedure)
				if err != nil {
					err = fmt.Errorf("imsi:%v, %v", imsiStr, err)
					pCtx.Log.Infoln("Result: FAIL,", err)
					QuitUe(pCtx)
					proc_fail = true
				} else if procedure == common.UNKNOWN_PROCEDURE {
					no_more_proc = true
				}
			case common.PROC_FAIL_EVENT:
				err = fmt.Errorf("imsi:%v, procedure:%v, error:%v", msg.Supi, msg.Proc, msg.Error)
				pCtx.Log.Infoln("Result: FAIL,", err)
				// most failures already tore the UE down through the event loop, but not those
				// that are reported straight to the profile (for example a failed connection to
				// the handover target); asking again is harmless
				QuitUe(pCtx)
				proc_fail = true
			}
		}
		ticker.Stop()
		if no_more_proc {
			pCtx.Log.Infoln("imsiStateMachine no more proc to execute")
			// the profile is done with this UE, so do not leave its workers running
			QuitUe(pCtx)
			break
		} else if proc_fail {
			break
		}
		// should we wait for pulse to move to next step?
		if profile.StepTrigger {
			pCtx.Log.Infoln("imsiStateMachine waiting for user trigger")
			msg, ok := <-pCtx.TrigEventsChan
			if ok {
				pCtx.Log.Infoln("imsiStateMachine received trigger:", msg)
			}
		}
	}
	pCtx.Log.Infoln("imsiStateMachine ended")
	return err
}
