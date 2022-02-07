// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package deregister

import (
	"gnbsim/common"
	"gnbsim/factory"
	profctx "gnbsim/profile/context"
	"gnbsim/profile/util"
	"gnbsim/simue"
	simuectx "gnbsim/simue/context"
	"strconv"
	"sync"
	"time"
)

func Deregister_test(profile *profctx.Profile) {
	initEventMap(profile)
	initProcedureList(profile)

	gnb, err := factory.AppConfig.Configuration.GetGNodeB(profile.GnbName)
	if err != nil {
		profile.Log.Errorln("GetGNodeB returned:", err)
	}

	imsi, err := strconv.Atoi(profile.StartImsi)
	if err != nil {
		profile.Log.Fatalln("invalid imsi value")
	}

	var wg sync.WaitGroup
	// Currently executing profile for one IMSI at a time
	for count := 1; count <= profile.UeCount; count++ {
		simUe := simuectx.NewSimUe("imsi-"+strconv.Itoa(imsi), gnb, profile)

		wg.Add(1)
		go simue.Init(simUe, &wg)
		util.SendToSimUe(simUe, common.PROFILE_START_EVENT)

		msg := <-profile.ReadChan
		switch msg.Event {
		case common.PROFILE_PASS_EVENT:
			profile.Log.Infoln("Result: PASS, SimUe:", msg.Supi)
		case common.PROFILE_FAIL_EVENT:
			profile.Log.Infoln("Result: FAIL, SimUe:", msg.Supi, "Failed Procedure:",
				msg.Proc, "Error:", msg.Error)
		}
		time.Sleep(2 * time.Second)
		imsi++
	}

	wg.Wait()
}

// initEventMap initializes the event map of profile with default values as per
// the procedures in the profile
func initEventMap(profile *profctx.Profile) {
	profile.Events = map[common.EventType]common.EventType{
		common.REG_REQUEST_EVENT:           common.AUTH_REQUEST_EVENT,
		common.AUTH_REQUEST_EVENT:          common.AUTH_RESPONSE_EVENT,
		common.SEC_MOD_COMMAND_EVENT:       common.SEC_MOD_COMPLETE_EVENT,
		common.REG_ACCEPT_EVENT:            common.REG_COMPLETE_EVENT,
		common.PDU_SESS_EST_REQUEST_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
		common.PDU_SESS_EST_ACCEPT_EVENT:   common.PDU_SESS_EST_ACCEPT_EVENT,
		common.DEREG_REQUEST_UE_ORIG_EVENT: common.DEREG_ACCEPT_UE_ORIG_EVENT,
		common.PROFILE_PASS_EVENT:          common.QUIT_EVENT,
	}
}

func initProcedureList(profile *profctx.Profile) {
	profile.Procedures = []common.ProcedureType{
		common.REGISTRATION_PROCEDURE,
		common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
		common.USER_DATA_PKT_GENERATION_PROCEDURE,
		common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
	}
}
