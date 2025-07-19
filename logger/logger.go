// SPDX-FileCopyrightText: 2024 Intel Corporation
// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log           *zap.Logger
	summaryLog    *zap.Logger
	AppLog        *zap.SugaredLogger
	AppSummaryLog *zap.SugaredLogger
	RealUeLog     *zap.SugaredLogger
	SimUeLog      *zap.SugaredLogger
	ProfileLog    *zap.SugaredLogger
	GNodeBLog     *zap.SugaredLogger
	CfgLog        *zap.SugaredLogger
	UtilLog       *zap.SugaredLogger
	GtpLog        *zap.SugaredLogger
	NgapLog       *zap.SugaredLogger
	PsuppLog      *zap.SugaredLogger
	GinLog        *zap.SugaredLogger
	HttpLog       *zap.SugaredLogger
	ProfUeCtxLog  *zap.SugaredLogger
	StatsLog      *zap.SugaredLogger
	atomicLevel   zap.AtomicLevel
)

const (
	FieldSupi        string = "supi"
	FieldProfile     string = "profile"
	FieldGnb         string = "gnb"
	FieldGnbUeNgapId string = "ranuengapid"
	FieldDlTeid      string = "dlteid"
	FieldPduSessId   string = "pdusessid"
	FieldIp          string = "ip"
)

func init() {
	atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)

	log = buildLogger("gnbsim.log")
	summaryLog = buildLogger("summary.log")

	base := log.Sugar().With("component", "GNBSIM")
	summary := summaryLog.Sugar().With("component", "GNBSIM")

	AppLog = base.With("category", "App")
	AppSummaryLog = summary.With("category", "Summary")
	RealUeLog = base.With("category", "RealUe")
	SimUeLog = base.With("category", "SimUe")
	ProfUeCtxLog = base.With("category", "ProfUeCtx")
	ProfileLog = base.With("category", "Profile")
	GNodeBLog = base.With("category", "GNodeB")
	GinLog = base.With("category", "Gin")
	HttpLog = base.With("category", "HTTP")
	CfgLog = base.With("category", "CFG")
	StatsLog = base.With("category", "Stats")
	UtilLog = base.With("category", "Util")
	GtpLog = UtilLog.With("subcategory", "GTP")
	NgapLog = UtilLog.With("subcategory", "NGAP")
	PsuppLog = UtilLog.With("subcategory", "PSUPP")
}

func buildLogger(outputFile string) *zap.Logger {
	cfg := zap.Config{
		Level:            atomicLevel,
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout", outputFile},
		ErrorOutputPaths: []string{"stderr"},
	}

	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncoderConfig.MessageKey = "message"
	cfg.EncoderConfig.StacktraceKey = ""

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func GetLogger() *zap.Logger {
	return log
}

// SetLogLevel: set the log level (panic|fatal|error|warn|info|debug)
func SetLogLevel(level zapcore.Level) {
	AppLog.Infoln("set log level:", level)
	atomicLevel.SetLevel(level)
}
