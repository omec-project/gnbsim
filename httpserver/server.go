// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/omec-project/gnbsim/factory"
	"github.com/omec-project/gnbsim/logger"
	profilerouter "github.com/omec-project/gnbsim/profile/httprouter"
	"github.com/omec-project/util/http2_util"
	utilLogger "github.com/omec-project/util/logger"
)

var (
	server   *http.Server
	serverMu sync.Mutex
)

const (
	logFile     string        = "http2.log"
	CTX_TIMEOUT time.Duration = 5

	// defaultBindAddr is used when no address is configured, so that the unauthenticated
	// control API is never exposed on every interface by omission.
	defaultBindAddr = "127.0.0.1"
	// readHeaderTimeout bounds how long a client may take to send request headers (slowloris).
	readHeaderTimeout = 10 * time.Second
)

func StartHttpServer() (err error) {
	router := utilLogger.NewGinWithZap(logger.GinLog)
	router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin", "Content-Length", "Content-Type", "User-Agent", "Referrer", "Host",
			"Token", "X-Requested-With",
		},
		ExposeHeaders: []string{"Content-Length"},
		// The API has no cookie or session based authentication, so credentials are not allowed.
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           86400,
	}))

	// Register routes
	profilerouter.AddService(router)

	config := factory.AppConfig.Configuration
	bindAddr := config.Server.IpAddr
	if bindAddr == "" {
		logger.AppLog.Warnf("httpServer.ipAddr not set, listening on %s only", defaultBindAddr)
		bindAddr = defaultBindAddr
	}
	serverAddr := net.JoinHostPort(bindAddr, config.Server.Port)
	srv, err := http2_util.NewServer(serverAddr, logFile, router)

	if srv == nil {
		logger.AppLog.Errorf("initialize HTTP server failed: %+v", err)
		return fmt.Errorf("failed to initialize http server, err: %v", err)
	}

	if err != nil {
		logger.AppLog.Warnf("initialize HTTP server: %+v", err)
		return fmt.Errorf("failed to initialize http server, err: %v", err)
	}

	srv.ReadHeaderTimeout = readHeaderTimeout
	serverMu.Lock()
	server = srv
	serverMu.Unlock()

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	serverMu.Lock()
	srv := server
	serverMu.Unlock()
	if srv == nil {
		logger.HttpLog.Warnln("HTTP server was not started, nothing to shut down")
		return
	}

	if err := srv.Shutdown(ctx); err != nil {
		logger.HttpLog.Errorf("server shutdown failed:%+v", err)
	}
}
