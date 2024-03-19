// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/omec-project/logger_util"
	"github.com/sirupsen/logrus"
)

var (
	log           *logrus.Logger
	summaryLog    *logrus.Logger
	AppLog        *logrus.Entry
	AppSummaryLog *logrus.Entry
	RealUeLog     *logrus.Entry
	SimUeLog      *logrus.Entry
	ProfileLog    *logrus.Entry
	GNodeBLog     *logrus.Entry
	CfgLog        *logrus.Entry
	UtilLog       *logrus.Entry
	GtpLog        *logrus.Entry
	NgapLog       *logrus.Entry
	PsuppLog      *logrus.Entry
	GinLog        *logrus.Entry
	HttpLog       *logrus.Entry
	ProfUeCtxLog  *logrus.Entry
	StatsLog      *logrus.Entry
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
	log = logrus.New()
	summaryLog = logrus.New()
	log.SetReportCaller(false)
	summaryLog.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder: []string{
			"component", "category", "subcategory",
			FieldProfile, FieldSupi, FieldGnb, FieldGnbUeNgapId,
		},
	}

	summaryLog.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
	}

	selfLogHook, err := logger_util.NewFileHook("gnbsim.log",
		os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
	if err == nil {
		log.Hooks.Add(selfLogHook)
	}

	summaryLogHook, err := logger_util.NewFileHook("summary.log",
		os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
	if err == nil {
		summaryLog.Hooks.Add(summaryLogHook)
	}

	AppLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "App"})
	AppSummaryLog = summaryLog.WithFields(logrus.Fields{"component": "GNBSIM", "category": "Summary"})
	RealUeLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "RealUe"})
	SimUeLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "SimUe"})
	ProfUeCtxLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "ProfUeCtx"})
	ProfileLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "Profile"})
	GNodeBLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "GNodeB"})
	GinLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "Gin"})
	HttpLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "HTTP"})
	CfgLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "CFG"})
	StatsLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "Stats"})
	UtilLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "Util"})
	GtpLog = UtilLog.WithField("subcategory", "GTP")
	NgapLog = UtilLog.WithField("subcategory", "NGAP")
	PsuppLog = UtilLog.WithField("subcategory", "PSUPP")
}

func SetLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		AppLog.Fatalln("Failed to parse log level:", err)
	}
	log.SetLevel(lvl)
}

func SetReportCaller(set bool) {
	log.SetReportCaller(set)
}
