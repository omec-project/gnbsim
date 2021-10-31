// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package register

import (
	"gnbsim/common"
	"gnbsim/factory"
	profctx "gnbsim/profile/context"
	"gnbsim/profile/util"
	"gnbsim/simue"
	simuectx "gnbsim/simue/context"
	"strconv"
	"time"
	// AJAY - Change required
)

func Register_test(profile *profctx.Profile) {
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

	// Currently executing profile for one IMSI at a time
	for count := 1; count <= profile.UeCount; count++ {
		simUe := simuectx.NewSimUe("imsi-"+strconv.Itoa(imsi), gnb, profile)
		go simue.Init(simUe)
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
}

// initEventMap initializes the event map of profile with default values as per
// the procedures in the profile
func initEventMap(profile *profctx.Profile) {
	profile.Events = map[common.EventType]common.EventType{
		common.REG_REQUEST_EVENT:     common.AUTH_REQUEST_EVENT,
		common.AUTH_REQUEST_EVENT:    common.AUTH_RESPONSE_EVENT,
		common.SEC_MOD_COMMAND_EVENT: common.SEC_MOD_COMPLETE_EVENT,
		common.REG_ACCEPT_EVENT:      common.REG_COMPLETE_EVENT,
	}
}

func initProcedureList(profile *profctx.Profile) {
	profile.Procedures = []common.ProcedureType{common.REGISTRATION_PROCEDURE}
}
