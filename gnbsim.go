// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" //Using package only for invoking initialization.
	"os"
	"sync"

	"github.com/omec-project/http2_util"
	"github.com/omec-project/logger_util"
	"github.com/gin-contrib/cors"
	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	"github.com/omec-project/gnbsim/gnodeb"
	"github.com/omec-project/gnbsim/logger"
	prof "github.com/omec-project/gnbsim/profile"
	profctx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/gnbsim/profile/httpserver"

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

	prof.InitializeAllProfiles()
	err := gnodeb.InitializeAllGnbs()
	if err != nil {
		logger.AppLog.Errorln("Failed to initialize gNodeBs:", err)
		return err
	}

	go ListenAndLogSummary()

	router := logger_util.NewGinWithLogrus(logger.GinLog)
	router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin", "Content-Length", "Content-Type", "User-Agent", "Referrer", "Host",
			"Token", "X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
		MaxAge:           86400,
	}))

	httpserver.AddService(router)
	server, err := http2_util.NewServer("127.0.0.1:8081", "./http2.log", router)

	if server == nil {
		logger.AppLog.Errorf("Initialize HTTP server failed: %+v", err)
		return fmt.Errorf("failed to initialize http server, err: %v", err)
	}

	if err != nil {
		logger.AppLog.Warnf("Initialize HTTP server: %+v", err)
		return fmt.Errorf("failed to initialize http server, err: %v", err)
	}

	serverScheme := "http"
	if serverScheme == "http" {
		err = server.ListenAndServe()
	} else if serverScheme == "https" {
		//	err = server.ListenAndServeTLS(util.AmfPemPath, util.AmfKeyPath)
	}

	if err != nil {
		logger.AppLog.Fatalf("HTTP server setup failed: %+v", err)
	}

	var profileWaitGrp sync.WaitGroup
	for _, profile := range config.Configuration.Profiles {
		if !profile.Enable {
			continue
		}
		profileWaitGrp.Add(1)

		go func(profileCtx *profctx.Profile) {
			defer profileWaitGrp.Done()
			go prof.ExecuteProfile(profileCtx, profctx.SummaryChan)
		}(profile)

		if config.Configuration.ExecInParallel == false {
			profileWaitGrp.Wait()
		}
	}
	if config.Configuration.ExecInParallel == true {
		profileWaitGrp.Wait()
	}

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

func ListenAndLogSummary() {
	for intfcMsg := range profctx.SummaryChan {
		if intfcMsg.GetEventType() == common.QUIT_EVENT {
			return
		}

		result := "TRUE"
		// Waiting for execution summary from profile routine
		msg, ok := intfcMsg.(*common.SummaryMessage)
		if !ok {
			logger.AppLog.Fatalln("Invalid Message Type")
		}

		logger.AppSummaryLog.Infoln("Profile Name:", msg.ProfileName, ", Profile Type:", msg.ProfileType)
		logger.AppSummaryLog.Infoln("Ue's Passed:", msg.UePassedCount, ", Ue's Failed:", msg.UeFailedCount)

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
