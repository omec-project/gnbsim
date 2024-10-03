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
	config := zap.Config{
		Level:            atomicLevel,
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout", "gnbsim.log"},
		ErrorOutputPaths: []string{"stderr"},
	}

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.StacktraceKey = ""

	var err error
	log, err = config.Build()
	if err != nil {
		panic(err)
	}

	configSummary := zap.Config{
		Level:            atomicLevel,
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout", "summary.log"},
		ErrorOutputPaths: []string{"stderr"},
	}

	configSummary.EncoderConfig.TimeKey = "timestamp"
	configSummary.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	configSummary.EncoderConfig.LevelKey = "level"
	configSummary.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	configSummary.EncoderConfig.CallerKey = "caller"
	configSummary.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	configSummary.EncoderConfig.MessageKey = "message"
	configSummary.EncoderConfig.StacktraceKey = ""

	summaryLog, err = configSummary.Build()
	if err != nil {
		panic(err)
	}

	AppLog = log.Sugar().With("component", "GNBSIM", "category", "App")
	AppSummaryLog = summaryLog.Sugar().With("component", "GNBSIM", "category", "Summary")
	RealUeLog = log.Sugar().With("component", "GNBSIM", "category", "RealUe")
	SimUeLog = log.Sugar().With("component", "GNBSIM", "category", "SimUe")
	ProfUeCtxLog = log.Sugar().With("component", "GNBSIM", "category", "ProfUeCtx")
	ProfileLog = log.Sugar().With("component", "GNBSIM", "category", "Profile")
	GNodeBLog = log.Sugar().With("component", "GNBSIM", "category", "GNodeB")
	GinLog = log.Sugar().With("component", "GNBSIM", "category", "Gin")
	HttpLog = log.Sugar().With("component", "GNBSIM", "category", "HTTP")
	CfgLog = log.Sugar().With("component", "GNBSIM", "category", "CFG")
	StatsLog = log.Sugar().With("component", "GNBSIM", "category", "Stats")
	UtilLog = log.Sugar().With("component", "GNBSIM", "category", "Util")
	GtpLog = UtilLog.With("subcategory", "GTP")
	NgapLog = UtilLog.With("subcategory", "NGAP")
	PsuppLog = UtilLog.With("subcategory", "PSUPP")
}

func GetLogger() *zap.Logger {
	return log
}

// SetLogLevel: set the log level (panic|fatal|error|warn|info|debug)
func SetLogLevel(level zapcore.Level) {
	AppLog.Infoln("set log level:", level)
	atomicLevel.SetLevel(level)
}
