// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"net"

	"github.com/omec-project/gnbsim/logger"

	amfctx "github.com/omec-project/amf/context"
	"github.com/omec-project/amf/factory"
	"github.com/omec-project/openapi/models"
	"github.com/sirupsen/logrus"
)

const NGAP_SCTP_PORT int = 38412

// GnbAmf holds the AMF context
type GnbAmf struct {
	/* Indicates wether NGSetup was successful or not*/
	NgSetupStatus bool
	AmfHostName   string `yaml:"hostName"`
	AmfIp         string `yaml:"ipAddr"`
	AmfName       string
	AmfPort       int `yaml:"port"`
	/* Relative AMF Capacity */
	RelCap          int64
	ServedGuamiList []models.Guami
	PlmnSupportList []factory.PlmnSupportItem
	/*Socket Connection*/
	Conn net.Conn

	/* logger */
	Log *logrus.Entry
}

func NewGnbAmf(ip string, port int) *GnbAmf {
	gnbAmf := &GnbAmf{}
	gnbAmf.AmfIp = ip
	gnbAmf.AmfPort = port
	gnbAmf.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbAmf",
		logger.FieldIp: gnbAmf.AmfIp})
	return gnbAmf
}

func (amf *GnbAmf) Init() {
	amf.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbAmf",
		logger.FieldIp: amf.AmfIp})
}

func (amf *GnbAmf) GetIpAddr() string {
	return amf.AmfIp
}

func (amf *GnbAmf) GetPort() int {
	return amf.AmfPort
}

func (amf *GnbAmf) SetAMFName(name string) {
	amf.AmfName = name
}

func (amf *GnbAmf) SetRelativeAMFCapacity(cap int64) {
	amf.RelCap = cap
}

func (amf *GnbAmf) SetNgSetupStatus(successfulOutcome bool) {
	// TODO Access to this either should not be concurrent or should be
	// synchronized
	amf.NgSetupStatus = successfulOutcome
}

func (amf *GnbAmf) GetNgSetupStatus() bool {
	// TODO Access to this either should not be concurrent or should be
	// synchronized
	return amf.NgSetupStatus
}

func NewServedGUAMIList() []models.Guami {
	return make([]models.Guami, 0, amfctx.MaxNumOfServedGuamiList)
}

func NewPlmnSupportList() []factory.PlmnSupportItem {
	return make([]factory.PlmnSupportItem, 0, amfctx.MaxNumOfPLMNs)
}
