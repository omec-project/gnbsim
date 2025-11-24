// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/omec-project/gnbsim/factory"
	"github.com/omec-project/gnbsim/logger"
	profilerouter "github.com/omec-project/gnbsim/profile/httprouter"
	"github.com/omec-project/util/http2_util"
	utilLogger "github.com/omec-project/util/logger"
)

var server *http.Server

const (
	logFile     string        = "http2.log"
	CTX_TIMEOUT time.Duration = 5
)

func StartHttpServer() (err error) {
	router := utilLogger.NewGinWithZap(logger.GinLog)
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

	// Register routes
	profilerouter.AddService(router)

	config := factory.AppConfig.Configuration
	serverAddr := config.Server.IpAddr + ":" + config.Server.Port
	server, err = http2_util.NewServer(serverAddr, logFile, router)

	if server == nil {
		logger.AppLog.Errorf("initialize HTTP server failed: %+v", err)
		return fmt.Errorf("failed to initialize http server, err: %v", err)
	}

	if err != nil {
		logger.AppLog.Warnf("initialize HTTP server: %+v", err)
		return fmt.Errorf("failed to initialize http server, err: %v", err)
	}

	serverScheme := "http"
	if serverScheme == "http" {
		err = server.ListenAndServe()
	}

	if err != nil {
		logger.AppLog.Errorln("HTTP server setup failed:", err)
	}

	logger.HttpLog.Infoln("Server shut down")
	return nil
}

func StopHttpServer() {
	logger.HttpLog.Infoln("Shutting down HTTP server")
	ctx, cancel := context.WithTimeout(context.Background(), CTX_TIMEOUT*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		logger.HttpLog.Fatalf("server shutdown failed:%+v", err)
	}
}
