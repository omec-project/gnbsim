// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Using package only for invoking initialization.
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	"github.com/omec-project/gnbsim/gnodeb"
	"github.com/omec-project/gnbsim/httpserver"
	"github.com/omec-project/gnbsim/logger"
	prof "github.com/omec-project/gnbsim/profile"
	profctx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/gnbsim/stats"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap/zapcore"
)

func main() {
	app := &cli.Command{}
	app.Name = "GNBSIM"
	app.Usage = "gnbsim --cfg <gnbsim_config_file.yaml>"
	app.Action = action
	app.Flags = getCliFlags()

	logger.AppLog.Infoln("app name:", app.Name)

	if err := app.Run(context.Background(), os.Args); err != nil {
		logger.AppLog.Errorln("failed to run GNBSIM:", err)
		return
	}
}

func action(ctx context.Context, c *cli.Command) error {
	cfg := c.String("cfg")

	absPath, err := filepath.Abs(cfg)
	if err != nil {
		logger.AppLog.Errorln(err)
		return err
	}

	if err = factory.InitConfigFactory(absPath); err != nil {
		logger.AppLog.Errorln("failed to initialize config factory:", err)
		return err
	}

	config := factory.AppConfig

	// Initiating a server for profiling
	if config.Configuration.GoProfile.Enable {
		go func() {
			endpt := fmt.Sprintf(":%v", config.Configuration.GoProfile.Port)
			logger.AppLog.Infoln("endpoint for profile server", endpt)
			err = http.ListenAndServe(endpt, nil)
			if err != nil {
				logger.AppLog.Errorln("failed to start profiling server")
			}
		}()
	}
	lvl, errLevel := zapcore.ParseLevel(config.Logger.LogLevel)
	if errLevel != nil {
		logger.AppLog.Errorln("can not parse input level")
	}
	logger.AppLog.Infoln("setting log level to:", lvl)
	logger.SetLogLevel(lvl)

	err = prof.InitializeAllProfiles()
	if err != nil {
		logger.AppLog.Errorln("failed to initialize Profiles:", err)
		return err
	}

	err = gnodeb.InitializeAllGnbs()
	if err != nil {
		logger.AppLog.Errorln("failed to initialize gNodeBs:", err)
		return err
	}

	go ListenAndLogSummary()

	var appWaitGrp sync.WaitGroup
	if config.Configuration.Server.Enable {
		appWaitGrp.Add(1)
		go func() {
			defer appWaitGrp.Done()
			err := httpserver.StartHttpServer()
			if err != nil {
				logger.AppLog.Infoln("StartHttpServer returned :", err)
			}
		}()

		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-signalChannel
			logger.AppLog.Infoln("StopHttpServer called")
			httpserver.StopHttpServer()
			logger.AppLog.Infoln("StopHttpServer returned ")
		}()
	}

	// This configuration enables running the configured profiles
	// when gnbsim is started. It is enabled by default. If we want no
	// profiles to run and gnbsim to wait for a command, then we
	// should disable this config.
	if config.Configuration.RunConfigProfilesAtStart {
		var profileWaitGrp sync.WaitGroup
		// start profile and wait for it to finish (success or failure)
		// Keep running gnbsim as long as profiles are not finished
		for _, profile := range config.Configuration.Profiles {
			if !profile.Enable {
				logger.AppLog.Errorln("disabled profileType ", profile.ProfileType)
				continue
			}
			profileWaitGrp.Add(1)

			prof.InitProfile(profile, profctx.SummaryChan)

			go func(profileCtx *profctx.Profile) {
				defer profileWaitGrp.Done()
				prof.ExecuteProfile(profileCtx, profctx.SummaryChan)
			}(profile)

			if !config.Configuration.ExecInParallel {
				profileWaitGrp.Wait()
			}
		}

		if config.Configuration.ExecInParallel {
			profileWaitGrp.Wait()
		}
	}

	appWaitGrp.Wait()

	// should be good enough to send pending packets out of socket and process events on channel
	time.Sleep(time.Second * 5)
	stats.DumpStats()
	// TODO: To be removed. Allowing summary logger to dump the logs
	time.Sleep(time.Second * 5)

	return nil
}

func getCliFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "cfg",
			Usage:    "gNBSim config file",
			Required: true,
		},
	}
}

// TODO: we don't keep track of how many profiles are started...
func ListenAndLogSummary() {
	for intfcMsg := range profctx.SummaryChan {
		// TODO: do we need this event ?
		if intfcMsg.GetEventType() == common.QUIT_EVENT {
			return
		}

		result := "PASS"
		// Waiting for execution summary from profile routine
		msg, ok := intfcMsg.(*common.SummaryMessage)
		if !ok {
			logger.AppLog.Fatalln("invalid Message Type")
		}

		logger.AppSummaryLog.Infof("Profile Name: %v, Profile Type: %v", msg.ProfileName, msg.ProfileType)
		logger.AppSummaryLog.Infof("Ue's Passed: %v, Ue's Failed: %v", msg.UePassedCount, msg.UeFailedCount)

		if len(msg.ErrorList) != 0 {
			result = "FAIL"
			logger.AppSummaryLog.Infoln("Profile Errors:")
			for _, err := range msg.ErrorList {
				logger.AppSummaryLog.Errorln(err)
			}
		}
		logger.AppSummaryLog.Infoln("Profile Status:", result)
	}
}
