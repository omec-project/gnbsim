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

func InitUE(imsiStr string, gnb *gnbctx.GNodeB, profile *profctx.Profile, result chan *common.ProfileMessage) chan common.InterfaceMessage {
	simUe := simuectx.NewSimUe(imsiStr, gnb, profile, result)
	Init(simUe) // Initialize simUE, realUE & wait for events
	return simUe.ReadChan
}

func Init(simUe *simuectx.SimUe) {
	err := ConnectToGnb(simUe)
	if err != nil {
		err = fmt.Errorf("failed to connect to gnodeb: %v", err)
		simUe.Log.Infoln("sent Profile Fail Event to Profile routine****:", err)
		SendToProfile(simUe, common.PROC_FAIL_EVENT, err)
		return
	}

	simUe.WaitGrp.Add(1)
	go func() {
		defer simUe.WaitGrp.Done()
		realue.Init(simUe.RealUe)
	}()

	go HandleEvents(simUe)
	simUe.Log.Infoln("SIM UE Init complete")
}

func ConnectToGnb(simUe *simuectx.SimUe) error {
	uemsg := common.UuMessage{}
	uemsg.Event = common.CONNECTION_REQUEST_EVENT
	uemsg.CommChan = simUe.ReadChan
	uemsg.Supi = simUe.Supi

	var err error
	gNb := simUe.GnB
	simUe.WriteGnbUeChan, err = gnodeb.RequestConnection(gNb, &uemsg)
	if err != nil {
		simUe.Log.Errorf("connecting to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
			gNb.GnbN2Ip, gNb.GnbN2Port)
		return err
	}

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

	targetChan, err := gnodeb.RegisterHandoverTarget(targetGnb, sourceGnbCpUe.AmfUeNgapId, simUe.ReadChan)
	if err != nil {
		return fmt.Errorf("failed to register target gNB: %w", err)
	}
	simUe.TargetWriteGnbUeChan = targetChan

	simUe.Log.Infof("pre-registered with target gNB: %v (AMF-UE-NGAP-ID: %v)",
		targetGnb.GnbName, sourceGnbCpUe.AmfUeNgapId)
	return nil
}

func HandleEvents(ue *simuectx.SimUe) {
	var err error
	for msg := range ue.ReadChan {
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
	ue.WriteRealUeChan <- msg
}

func SendToGnbUe(ue *simuectx.SimUe, msg common.InterfaceMessage) {
	ue.Log.Debugln("sending", msg.GetEventType(), "to GnbUe")
	ue.WriteGnbUeChan <- msg
}

func SendToProfile(ue *simuectx.SimUe, event common.EventType, errMsg error) {
	ue.Log.Debugln("sending", event, "to Profile routine")
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Supi = ue.Supi
	msg.Proc = ue.Procedure
	msg.Error = errMsg
	ue.WriteProfileChan <- msg
	ue.Log.Debugln("sent ", event, "to Profile routine")
}

func RunProcedure(simUe *simuectx.SimUe, procedure common.ProcedureType) {
	util.SendToSimUe(simUe, common.PROC_START_EVENT, procedure)
}

func ImsiStateMachine(profile *profctx.Profile, pCtx *profctx.ProfileUeContext, imsiStr string, summaryChan chan common.InterfaceMessage) error {
	var no_more_proc bool
	var proc_fail bool
	var err error

	procedure := profile.GetNextProcedure(pCtx, 0)
	for {
		// select procedure to execute for imsi
		simUe := simuectx.GetSimUe(imsiStr)
		//if simUe == nil {
		// pass readChan to simUe
		//}
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
			proc_fail = true
		case msg := <-pCtx.ReadChan:
			pCtx.Log.Infoln("imsiStateMachine received result")
			end_ts := time.Now()
			diff := end_ts.Sub(start_ts)
			pCtx.Log.Infof("procedure: %v, status: %v, E2E latency [ms]: %v", procedure.String(), msg.GetEventType().String(), diff.Milliseconds())

			switch msg.Event {
			case common.PROC_PASS_EVENT:
				pCtx.Log.Infoln("Procedure Result: PASS, imsi:", msg.Supi)
				procedure = profile.GetNextProcedure(pCtx, simUe.Procedure)
				if procedure == 0 {
					no_more_proc = true
				}
			case common.PROC_FAIL_EVENT:
				err = fmt.Errorf("imsi:%v, procedure:%v, error:%v", msg.Supi, msg.Proc, msg.Error)
				pCtx.Log.Infoln("Result: FAIL,", err)
				proc_fail = true
			}
		}
		ticker.Stop()
		if no_more_proc {
			pCtx.Log.Infoln("imsiStateMachine no more proc to execute")
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
