// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"github.com/free5gc/logger_util"
)

var (
	log       *logrus.Logger
	AppLog    *logrus.Entry
	RealUeLog *logrus.Entry
	SimUeLog  *logrus.Entry
)

const (
	FieldSupi string = "supi"
)

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category", "subcategory", FieldSupi},
	}

	selfLogHook, err := logger_util.NewFileHook("gnbsim.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
	if err == nil {
		log.Hooks.Add(selfLogHook)
	}

	AppLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "App"})
	RealUeLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "RealUe"})
	SimUeLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "SimUe"})
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(set bool) {
	log.SetReportCaller(set)
}
