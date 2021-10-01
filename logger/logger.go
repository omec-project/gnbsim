// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/free5gc/logger_util"
	"github.com/sirupsen/logrus"
)

var (
	log        *logrus.Logger
	AppLog     *logrus.Entry
	RealUeLog  *logrus.Entry
	SimUeLog   *logrus.Entry
	ProfileLog *logrus.Entry
	GNodeBLog  *logrus.Entry
	CfgLog     *logrus.Entry
)

const (
	FieldSupi        string = "supi"
	FieldProfile     string = "profile"
	FieldGnb         string = "gnb"
	FieldGnbUeNgapId string = "ranuengapid"
	FieldDlTeid      string = "dlteid"
	FieldIp          string = "ip"
)

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder: []string{"component", "category", "subcategory",
			FieldProfile, FieldSupi, FieldGnb, FieldGnbUeNgapId},
	}

	selfLogHook, err := logger_util.NewFileHook("gnbsim.log",
		os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
	if err == nil {
		log.Hooks.Add(selfLogHook)
	}

	AppLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "App"})
	RealUeLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "RealUe"})
	SimUeLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "SimUe"})
	ProfileLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "Profile"})
	GNodeBLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "GNodeB"})
	CfgLog = log.WithFields(logrus.Fields{"component": "GNBSIM", "category": "CFG"})

	SetLogLevel(logrus.TraceLevel)
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(set bool) {
	log.SetReportCaller(set)
}
