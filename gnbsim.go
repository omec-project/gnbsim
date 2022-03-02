// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"gnbsim/common"
	"gnbsim/factory"
	"gnbsim/gnodeb"
	"gnbsim/logger"
	"gnbsim/profile"
	"net/http"
	_ "net/http/pprof" //Using package only for invoking initialization.
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "GNBSIM"
	app.Usage = "./gnbsim -cfg [gnbsim configuration file]"
	app.Action = action
	app.Flags = getCliFlags()

	logger.AppLog.Infoln("App Name:", app.Name)

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Errorln("Failed to run GNBSIM:", err)
		return
	}
}

func action(c *cli.Context) error {
	cfg := c.String("cfg")
	if cfg == "" {
		logger.AppLog.Warnln("No configuration file provided. Using default configuration file:", factory.GNBSIM_DEFAULT_CONFIG_PATH)
		logger.AppLog.Infoln("Application Usage:", c.App.Usage)
		cfg = factory.GNBSIM_DEFAULT_CONFIG_PATH
	}

	if err := factory.InitConfigFactory(cfg); err != nil {
		logger.AppLog.Errorln("Failed to initialize config factory:", err)
		return err
	}

	//Initiating a server for profiling
	go func() {
		err := http.ListenAndServe(":5000", nil)
		if err != nil {
			logger.AppLog.Errorln("Failed to start profiling server")
		}
	}()

	config := factory.AppConfig
	lvl := config.Logger.LogLevel
	logger.AppLog.Infoln("Setting log level to:", lvl)
	logger.SetLogLevel(lvl)

	profile.InitializeAllProfiles()
	err := gnodeb.InitializeAllGnbs()
	if err != nil {
		logger.AppLog.Errorln("Failed to initialize gNodeBs:", err)
		return err
	}

	summaryChan := make(chan common.InterfaceMessage)
	result := "PASS"

	for _, profileCtx := range config.Configuration.Profiles {
		if profileCtx.Enable {
			logger.AppLog.Infoln("executing profile:", profileCtx.Name,
				", profile type:", profileCtx.ProfileType)

			go profile.ExecuteProfile(profileCtx, summaryChan)

			// Waiting for execution summary from profile routine
			msg, ok := (<-summaryChan).(*common.SummaryMessage)
			if !ok {
				logger.AppLog.Fatalln("Invalid Message Type")
			}

			logger.AppSummaryLog.Infoln("Profile Name:", msg.ProfileName, ", Profile Type:", msg.ProfileType)
			logger.AppSummaryLog.Infoln("Ue's Passed:", msg.UePassedCount, ", Ue's Failed:", msg.UeFailedCount)

			if msg.UeFailedCount != 0 {
				result = "FAIL"
			}

			if len(msg.ErrorList) != 0 {
				logger.AppSummaryLog.Infoln("Profile Errors:")
				for _, err := range msg.ErrorList {
					logger.AppSummaryLog.Errorln(err)
				}
			}
		}
	}

	logger.AppSummaryLog.Infoln("Simulation Result:", result)

	return nil
}

func getCliFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "cfg",
			Usage: "GNBSIM config file",
		},
	}
}
