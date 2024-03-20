// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/logger"
	profctx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/gnbsim/simue"
)

const IMSI_PREFIX = "imsi-"

func InitializeAllProfiles() error {
	for _, profile := range factory.AppConfig.Configuration.Profiles {
		err := profile.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func InitProfile(profile *profctx.Profile, summaryChan chan common.InterfaceMessage) {
	summary := &common.SummaryMessage{
		ProfileType: profile.ProfileType,
		ProfileName: profile.Name,
		ErrorList:   make([]error, 0, 10),
	}

	imsi, err := strconv.Atoi(profile.StartImsi)
	if err != nil {
		err = fmt.Errorf("invalid imsi value:%v", profile.StartImsi)
		summary.ErrorList = append(summary.ErrorList, err)
		summaryChan <- summary
		return
	}

	startImsi := imsi
	profile.Imsi = imsi

	profile.Log.Infoln("Init profile:", profile.Name,
		", profile type:", profile.ProfileType)

	gnb, err := factory.AppConfig.Configuration.GetGNodeB(profile.GnbName)
	if err != nil {
		err = fmt.Errorf("Failed to fetch gNB context: %v", err)
		summary.ErrorList = append(summary.ErrorList, err)
		summaryChan <- summary
		return
	}

	for count := 1; count <= profile.UeCount; count++ {
		imsiStr := makeImsiStr(profile, startImsi)
		initImsi(profile, gnb, imsiStr)
		startImsi++
	}
}

// makeImsiStr constructs IMSI string with specified integer value and proper length.
func makeImsiStr(profile *profctx.Profile, imsi int) string {
	s := strconv.Itoa(imsi)
	return IMSI_PREFIX + strings.Repeat("0", max(0, len(profile.StartImsi)-len(s))) + s
}

func initImsi(profile *profctx.Profile, gnb *gnbctx.GNodeB, imsiStr string) {
	readChan := make(chan *common.ProfileMessage)
	c := simue.InitUE(imsiStr, gnb, profile, readChan)
	p := profctx.ProfileUeContext{WriteSimChan: c}
	p.CurrentItr = profile.StartIteration
	p.ReadChan = readChan
	trigChan := make(chan *common.ProfileMessage)
	p.TrigEventsChan = trigChan
	p.Log = logger.ProfUeCtxLog.WithField(logger.FieldSupi, imsiStr)
	profile.PSimUe[imsiStr] = &p
}

// option1 : Run default profile start to end..Once done Received
//    - PROFILE_PASS
//    - PROFILE_FAIL
// option2 : Run custom profile start to end thorugh iterations. Once done receive
//    - PROFILE_PASS
//    - PROFILE_FAIL
//    - PROFILE_HOLD optionally, if next iteration is hold
// option3 : Pulses to come from REST Api
//    - Start, End, Run-x, Run-y, runx/imsi1, rumy/imsi2)
//    - Hold the calls
//    - We should be able to pass events to profile

func ExecuteProfile(profile *profctx.Profile, summaryChan chan common.InterfaceMessage) {
	profile.Log.Infoln("ExecuteProfile started ")
	var wg sync.WaitGroup
	var Mu sync.Mutex

	summary := &common.SummaryMessage{
		ProfileType: profile.ProfileType,
		ProfileName: profile.Name,
		ErrorList:   make([]error, 0, 10),
	}

	defer func() {
		summaryChan <- summary
	}()

	go func() {
		var plock sync.Mutex
		for msg := range profile.ReadChan {
			profile.Log.Infoln("Received trigger for profile ", msg)
			// works only if profile is still running.
			// Typically if execInParallel set true in profile
			gnb, err := factory.AppConfig.Configuration.GetGNodeB(profile.GnbName)
			if err != nil {
				err = fmt.Errorf("failed to fetch gNB context: %v", err)
				summary.ErrorList = append(summary.ErrorList, err)
				summaryChan <- summary
				return
			}

			plock.Lock()
			profile.UeCount = profile.UeCount + 1
			imsi := profile.Imsi + profile.UeCount
			imsiStr := makeImsiStr(profile, imsi)
			initImsi(profile, gnb, imsiStr)
			pCtx := profile.PSimUe[imsiStr]
			profile.Log.Infoln("pCtx ", pCtx)
			wg.Add(1)
			go func(pCtx *profctx.ProfileUeContext) {
				defer wg.Done()
				err := simue.ImsiStateMachine(profile, pCtx, imsiStr, summaryChan)
				// Execution for the UE is complete. Count UE result as success or failure
				Mu.Lock()
				if err != nil {
					summary.UeFailedCount++
					summary.ErrorList = append(summary.ErrorList, err)
				} else {
					summary.UePassedCount++
				}
				Mu.Unlock()
			}(pCtx)
			plock.Unlock()
		}
	}()
	imsi := profile.Imsi
	for count := 1; count <= profile.UeCount; count++ {
		imsiStr := makeImsiStr(profile, imsi)
		imsi++
		wg.Add(1)
		pCtx := profile.PSimUe[imsiStr]

		go func(pCtx *profctx.ProfileUeContext) {
			defer wg.Done()
			err := simue.ImsiStateMachine(profile, pCtx, imsiStr, summaryChan)
			// Execution for the UE is complete. Count UE result as success or failure
			Mu.Lock()
			if err != nil {
				summary.UeFailedCount++
				summary.ErrorList = append(summary.ErrorList, err)
			} else {
				summary.UePassedCount++
			}
			Mu.Unlock()
		}(pCtx)

		if !profile.ExecInParallel {
			profile.Log.Infoln("ExecuteProfile ExecInParallel false. Waiting for UEs to finish procesessing")
			wg.Wait()
		}
	}
	if profile.ExecInParallel {
		profile.Log.Infoln("ExecuteProfile ExecInParallel true. Waiting for for all UEs to finish processing")
		wg.Wait()
	}
	profile.Log.Infoln("ExecuteProfile ended")
}

// enable step trigger only if execParallel is enabled in profile
func SendStepEventProfile(name string) error {
	profile, found := profctx.ProfileMap[name]
	if !found {
		err := fmt.Errorf("unknown profile:%+v", profile)
		log.Println(err)
		return err
	}
	if !profile.ExecInParallel {
		err := fmt.Errorf("ExecInParallel should be true if step profile needs to be executed")
		log.Println(err)
		return err
	}
	msg := &common.ProfileMessage{}
	// msg.Supi =
	// msg.ProcedureType =
	msg.Event = common.PROFILE_STEP_EVENT
	for _, ctx := range profile.PSimUe {
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - start")
		ctx.TrigEventsChan <- msg
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - end")
	}
	return nil
}

func SendAddNewCallsEventProfile(name string, number int32) error {
	profile, found := profctx.ProfileMap[name]
	if !found {
		err := fmt.Errorf("unknown profile:%+v", profile)
		return err
	}
	msg := &common.ProfileMessage{}
	msg.Event = common.PROFILE_ADDCALLS_EVENT
	var i int32
	for i = 0; i < number; i++ {
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - start")
		profile.ReadChan <- msg
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - end")
	}
	return nil
}
