// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/gnodeb"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	profctx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/gnbsim/profile/util"
	"github.com/omec-project/gnbsim/realue"
	simuectx "github.com/omec-project/gnbsim/simue/context"
)

func RunProfile(imsiStr string, gnb *gnbctx.GNodeB, profile *profctx.Profile) error {
	var err error
	var cont bool
	var nextItr string = "quit"

	for {
		simUe := simuectx.NewSimUe(imsiStr, gnb, profile)
		simUe.CurrentItr = nextItr

		Init(simUe) // Initialize simUE, realUE & wait for events

		if simUe.CurrentItr != "quit" {
			simUe.Log.Infoln("Sent Profile continue Event ", simUe.CurrentItr)
			util.SendToSimUe(simUe, common.PROFILE_CONT_EVENT)
		} else {
			util.SendToSimUe(simUe, common.PROFILE_START_EVENT)
		}

		timeout := time.Duration(profile.PerUserTimeout) * time.Second
		ticker := time.NewTicker(timeout)

		select {
		case <-ticker.C:
			err = fmt.Errorf("imsi:%v, profile timeout", imsiStr)
			profile.Log.Infoln("Result: FAIL,", err)
			util.SendToSimUe(simUe, common.QUIT_EVENT)

		case msg := <-profile.ReadChan:
			switch msg.Event {
			case common.PROFILE_PASS_EVENT:
				profile.Log.Infoln("Result: PASS, imsi:", msg.Supi)
				cont = false
			case common.PROFILE_CONT_EVENT:
				profile.Log.Infoln("Result: continue ", msg.Supi)
				nextItr = msg.NextItr
				cont = true
			case common.PROFILE_FAIL_EVENT:
				err := fmt.Errorf("imsi:%v, procedure:%v, error:%v", msg.Supi, msg.Proc, msg.Error)
				profile.Log.Infoln("Result: FAIL,", err)
				cont = false
			}
		}
		ticker.Stop()
		time.Sleep(2 * time.Second)
		if cont == false {
			return err
		}
	}
	return err
}

func Init(simUe *simuectx.SimUe) {

	err := ConnectToGnb(simUe)
	if err != nil {
		err = fmt.Errorf("failed to connect to gnodeb: %v", err)
		SendToProfile(simUe, common.PROFILE_FAIL_EVENT, err)
		simUe.Log.Infoln("Sent Profile Fail Event to Profile routine")
		return
	}

	simUe.WaitGrp.Add(1)
	go func() {
		defer simUe.WaitGrp.Done()
		realue.Init(simUe.RealUe)
	}()

	go HandleEvents(simUe)
	simUe.Log.Infoln("SIM UE go routine complete")
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
		return err
	}

	simUe.Log.Infof("Connected to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
		gNb.GnbN2Ip, gNb.GnbN2Port)
	return nil
}

func HandleEvents(ue *simuectx.SimUe) {
	var err error
	for msg := range ue.ReadChan {
		event := msg.GetEventType()
		ue.Log.Infoln("Handling event:", event)

		switch event {
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
		case common.PROFILE_START_EVENT:
			err = HandleProfileStartEvent(ue, msg)
		case common.PROFILE_CONT_EVENT:
			err = HandleProfileContinueEvent(ue, msg)
		case common.CONNECTION_RELEASE_REQUEST_EVENT:
			err = HandleConnectionReleaseRequestEvent(ue, msg)
		case common.DEREG_REQUEST_UE_TERM_EVENT:
			err = HandleNwDeregRequestEvent(ue, msg)
		case common.DEREG_ACCEPT_UE_TERM_EVENT:
			err = HandleNwDeregAcceptEvent(ue, msg)
		case common.ERROR_EVENT:
			HandleErrorEvent(ue, msg)
			return
		case common.QUIT_EVENT:
			HandleQuitEvent(ue, msg)
			return
		default:
			ue.Log.Warnln("Event:", event, "is not supported")
		}

		if err != nil {
			ue.Log.Errorln("Failed to handle event:", event, "Error:", err)
			msg := &common.UeMessage{}
			msg.Error = err
			err = nil
			msg.Event = common.ERROR_EVENT
			HandleErrorEvent(ue, msg)
			return
		}
	}

	return
}

func SendToRealUe(ue *simuectx.SimUe, msg common.InterfaceMessage) {
	ue.Log.Traceln("Sending", msg.GetEventType(), "to RealUe")
	ue.WriteRealUeChan <- msg
}

func SendToGnbUe(ue *simuectx.SimUe, msg common.InterfaceMessage) {
	ue.Log.Traceln("Sending", msg.GetEventType(), "to GnbUe")
	ue.WriteGnbUeChan <- msg
}

func SendToProfile(ue *simuectx.SimUe, event common.EventType, errMsg error) {
	ue.Log.Traceln("Sending", event, "to Profile routine")
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Supi = ue.Supi
	msg.Proc = ue.Procedure
	if event == common.PROFILE_CONT_EVENT {
		msg.NextItr = errMsg.Error()
	} else {
		msg.Error = errMsg
	}
	ue.WriteProfileChan <- msg
	ue.Log.Traceln("Sent ", event, "to Profile routine")
}
