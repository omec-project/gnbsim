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
	simuectx "github.com/omec-project/gnbsim/simue/context"
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

// InitProfile creates the SimUe contexts of a profile. Failures are also reported on
// summaryChan; the returned error lets the caller refuse to execute a half-initialised profile.
func InitProfile(profile *profctx.Profile, summaryChan chan common.InterfaceMessage) error {
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
		return err
	}

	startImsi := imsi
	profile.Imsi = imsi

	profile.Log.Infoln("Init profile:", profile.Name,
		", profile type:", profile.ProfileType)

	gnb, err := factory.AppConfig.Configuration.GetGNodeB(profile.GnbName)
	if err != nil {
		err = fmt.Errorf("failed to fetch gNB context: %v", err)
		summary.ErrorList = append(summary.ErrorList, err)
		summaryChan <- summary
		return err
	}

	if profile.UeCount < 0 {
		err = fmt.Errorf("invalid ueCount value:%v", profile.UeCount)
		summary.ErrorList = append(summary.ErrorList, err)
		summaryChan <- summary
		return err
	}

	created := make([]string, 0, profile.UeCount)
	for count := 1; count <= profile.UeCount; count++ {
		imsiStr := makeImsiStr(profile, startImsi)
		if err = initImsi(profile, gnb, imsiStr); err != nil {
			err = fmt.Errorf("imsi:%v, failed to initialize UE: %v", imsiStr, err)
			summary.ErrorList = append(summary.ErrorList, err)
			summaryChan <- summary
			// the profile will not execute, so stop the UEs that were already started
			for _, started := range created {
				releaseImsi(profile, started)
			}
			return err
		}
		created = append(created, imsiStr)
		startImsi++
	}
	// only now is the profile complete enough for step and add-calls requests to find it
	profile.Publish()
	return nil
}

// makeImsiStr constructs IMSI string with specified integer value and proper length.
func makeImsiStr(profile *profctx.Profile, imsi int) string {
	s := strconv.Itoa(imsi)
	return IMSI_PREFIX + strings.Repeat("0", max(0, len(profile.StartImsi)-len(s))) + s
}

// initImsi creates and starts the UE. The IMSI is reserved until the SimUe has terminated, so a
// second profile cannot start an IMSI whose previous UE still has live workers.
func initImsi(profile *profctx.Profile, gnb *gnbctx.GNodeB, imsiStr string) error {
	if !simuectx.ReserveSupi(imsiStr) {
		return fmt.Errorf("already in use by a running profile")
	}
	readChan := make(chan *common.ProfileMessage)
	simUe, err := simue.InitUE(imsiStr, gnb, profile, readChan)
	if err != nil {
		simuectx.ReleaseSupi(imsiStr)
		return err
	}
	p := profctx.ProfileUeContext{WriteSimChan: simUe.ReadChan, Quit: simUe.RequestQuit, Terminated: simUe.Terminated()}
	p.CurrentItr = profile.StartIteration
	p.ReadChan = readChan
	trigChan := make(chan *common.ProfileMessage)
	p.TrigEventsChan = trigChan
	p.Log = logger.ProfUeCtxLog.With(logger.FieldSupi, imsiStr)
	profile.SetUe(imsiStr, &p)
	return nil
}

// releaseImsi stops a UE that was initialised but will not be executed, which runs the SimUe
// teardown path, and waits for it so that the IMSI is free when this returns.
func releaseImsi(profile *profctx.Profile, imsiStr string) {
	if pCtx := profile.GetUe(imsiStr); pCtx != nil {
		simue.QuitUeAndWait(pCtx)
	}
	profile.DeleteUe(imsiStr)
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
	profile.Log.Infoln("ExecuteProfile started")
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

	// UEs added while the profile runs are started by the goroutine below, which also counts them
	// in UeCount; this loop must only start the ones that existed when it began, or it would start
	// them a second time, and it would read UeCount while that goroutine writes it.
	initialUeCount := profile.UeCount

	go func() {
		var plock sync.Mutex
		for msg := range profile.ReadChan {
			profile.Log.Infoln("received trigger for profile", msg)
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
			if err = initImsi(profile, gnb, imsiStr); err != nil {
				profile.UeCount--
				plock.Unlock()
				err = fmt.Errorf("imsi:%v, failed to initialize UE: %v", imsiStr, err)
				Mu.Lock()
				summary.UeFailedCount++
				summary.ErrorList = append(summary.ErrorList, err)
				Mu.Unlock()
				continue
			}
			pCtx := profile.GetUe(imsiStr)
			profile.Log.Infoln("pCtx", pCtx)
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
	for count := 1; count <= initialUeCount; count++ {
		imsiStr := makeImsiStr(profile, imsi)
		imsi++
		pCtx := profile.GetUe(imsiStr)
		if pCtx == nil {
			// the profile was not initialised for this UE (see InitProfile)
			Mu.Lock()
			summary.UeFailedCount++
			summary.ErrorList = append(summary.ErrorList, fmt.Errorf("imsi:%v, UE context not initialised", imsiStr))
			Mu.Unlock()
			continue
		}
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
	profile, found := profctx.GetProfile(name)
	if !found {
		err := fmt.Errorf("unknown profile:%v", name)
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
	// snapshot so that UEs added meanwhile cannot race with the iteration
	for _, ctx := range profile.Ues() {
		profile.Log.Debugln("profile", profile.Name, ", writing on trig channel - start")
		ctx.TrigEventsChan <- msg
		profile.Log.Debugln("profile", profile.Name, ", writing on trig channel - end")
	}
	return nil
}

func SendAddNewCallsEventProfile(name string, number int32) error {
	profile, found := profctx.GetProfile(name)
	if !found {
		err := fmt.Errorf("unknown profile:%v", name)
		return err
	}
	msg := &common.ProfileMessage{}
	msg.Event = common.PROFILE_ADDCALLS_EVENT
	var i int32
	for i = 0; i < number; i++ {
		profile.Log.Debugln("profile", profile.Name, ", writing on trig channel - start")
		profile.ReadChan <- msg
		profile.Log.Debugln("profile", profile.Name, ", writing on trig channel - end")
	}
	return nil
}
