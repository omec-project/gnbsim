// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	profctx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/gnbsim/simue"
)

//profile names
const (
	REGISTER                string = "register"
	PDU_SESS_EST            string = "pdusessest"
	DEREGISTER              string = "deregister"
	AN_RELEASE              string = "anrelease"
	UE_TRIGG_SERVICE_REQ    string = "uetriggservicereq"
	NW_TRIGG_UE_DEREG       string = "nwtriggeruedereg"
	UE_REQ_PDU_SESS_RELEASE string = "uereqpdusessrelease"
	NW_REQ_PDU_SESS_RELEASE string = "nwreqpdusessrelease"
)

func InitializeAllProfiles() {
	for _, profile := range factory.AppConfig.Configuration.Profiles {
		profile.Init()
	}
	initProcedureEventMap()
}

func ExecuteProfile(profile *profctx.Profile, summaryChan chan common.InterfaceMessage) {

	summary := &common.SummaryMessage{
		ProfileType: profile.ProfileType,
		ProfileName: profile.Name,
		ErrorList:   make([]error, 0, 10),
	}

	defer func() {
		summaryChan <- summary
	}()

	err := initProcedureList(profile)
	if err != nil {
		summary.ErrorList = append(summary.ErrorList, err)
		return
	}

	gnb, err := factory.AppConfig.Configuration.GetGNodeB(profile.GnbName)
	if err != nil {
		err = fmt.Errorf("Failed to fetch gNB context: %v", err)
		summary.ErrorList = append(summary.ErrorList, err)
		return
	}

	imsi, err := strconv.Atoi(profile.StartImsi)
	if err != nil {
		err = fmt.Errorf("invalid imsi value:%v", profile.StartImsi)
		summary.ErrorList = append(summary.ErrorList, err)
		return
	}

	profile.Log.Infoln("executing profile:", profile.Name,
		", profile type:", profile.ProfileType)

	if profile.PerUserTimeout == 0 {
		profile.PerUserTimeout = profctx.PER_USER_TIMEOUT
	}

	var wg sync.WaitGroup
	var Mu sync.Mutex

	for count := 1; count <= profile.UeCount; count++ {
		imsiStr := "imsi-" + strconv.Itoa(imsi)
		imsi++
		wg.Add(1)
		go func(imsiStr string) {
			defer wg.Done()
			err = simue.RunProfile(imsiStr, gnb, profile)
			Mu.Lock()
			if err != nil {
				summary.UeFailedCount++
				summary.ErrorList = append(summary.ErrorList, err)
			} else {
				summary.UePassedCount++
			}
			Mu.Unlock()
		}(imsiStr)
		if profile.ExecInParallel == false {
			wg.Wait()
		}
	}
	if profile.ExecInParallel == true {
		wg.Wait()
	}
}

func initProcedureEventMap() {
	proc1 := profctx.ProcedureEventsDetails{}

	//common.REGISTRATION_PROCEDURE:
	proc1.Events = map[common.EventType]common.EventType{
		common.REG_REQUEST_EVENT:     common.AUTH_REQUEST_EVENT,
		common.AUTH_REQUEST_EVENT:    common.AUTH_RESPONSE_EVENT,
		common.SEC_MOD_COMMAND_EVENT: common.SEC_MOD_COMPLETE_EVENT,
		common.REG_ACCEPT_EVENT:      common.REG_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:    common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.REGISTRATION_PROCEDURE] = &proc1

	// common.PDU_SESSION_ESTABLISHMENT_PROCEDURE:
	proc2 := profctx.ProcedureEventsDetails{}
	proc2.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_EST_REQUEST_EVENT: common.PDU_SESS_EST_ACCEPT_EVENT,
		common.PDU_SESS_EST_ACCEPT_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.PDU_SESSION_ESTABLISHMENT_PROCEDURE] = &proc2

	//common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
	proc3 := profctx.ProcedureEventsDetails{}
	proc3.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_REL_REQUEST_EVENT: common.PDU_SESS_REL_COMMAND_EVENT,
		common.PDU_SESS_REL_COMMAND_EVENT: common.PDU_SESS_REL_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE] = &proc3

	// common.UE_INITIATED_DEREGISTRATION_PROCEDURE:
	proc4 := profctx.ProcedureEventsDetails{}
	proc4.Events = map[common.EventType]common.EventType{
		common.DEREG_REQUEST_UE_ORIG_EVENT: common.DEREG_ACCEPT_UE_ORIG_EVENT,
		common.PROFILE_PASS_EVENT:          common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.UE_INITIATED_DEREGISTRATION_PROCEDURE] = &proc4

	// common.AN_RELEASE_PROCEDURE:
	proc5 := profctx.ProcedureEventsDetails{}
	proc5.Events = map[common.EventType]common.EventType{
		common.TRIGGER_AN_RELEASE_EVENT: common.CONNECTION_RELEASE_REQUEST_EVENT,
		common.PROFILE_PASS_EVENT:       common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.AN_RELEASE_PROCEDURE] = &proc5

	// common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE:
	proc6 := profctx.ProcedureEventsDetails{}
	proc6.Events = map[common.EventType]common.EventType{
		common.SERVICE_REQUEST_EVENT: common.SERVICE_ACCEPT_EVENT,
		common.PROFILE_PASS_EVENT:    common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE] = &proc6

	// common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE:
	proc7 := profctx.ProcedureEventsDetails{}
	proc7.Events = map[common.EventType]common.EventType{
		common.DEREG_REQUEST_UE_TERM_EVENT: common.DEREG_ACCEPT_UE_TERM_EVENT,
		common.PROFILE_PASS_EVENT:          common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE] = &proc7

	// common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
	proc8 := profctx.ProcedureEventsDetails{}
	proc8.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_REL_COMMAND_EVENT: common.PDU_SESS_REL_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE] = &proc8

	// common.USER_DATA_PKT_GENERATION_PROCEDURE:
	proc9 := profctx.ProcedureEventsDetails{}
	proc9.Events = map[common.EventType]common.EventType{
		common.PROFILE_PASS_EVENT: common.QUIT_EVENT,
	}
	profctx.ProceduresMap[common.USER_DATA_PKT_GENERATION_PROCEDURE] = &proc9
}

func initProcedureList(profile *profctx.Profile) error {
	switch profile.ProfileType {
	case REGISTER:
		profile.Procedures = []common.ProcedureType{common.REGISTRATION_PROCEDURE}
	case PDU_SESS_EST:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
		}
	case DEREGISTER:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
		}
	case AN_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.AN_RELEASE_PROCEDURE,
		}
	case UE_TRIGG_SERVICE_REQ:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.AN_RELEASE_PROCEDURE,
			common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
		}
	case NW_TRIGG_UE_DEREG:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE,
		}
	case UE_REQ_PDU_SESS_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE,
		}
	case NW_REQ_PDU_SESS_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE,
		}
	default:
		// Custom Profiles do not have prefdefined procedure list
		return nil
	}
	return nil
}
