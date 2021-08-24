// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"net"

	"github.com/free5gc/amf/context"
	"github.com/free5gc/amf/factory"
	"github.com/free5gc/openapi/models"
)

//TODO Shift this to context package within gnodeb

// GnbAmf holds the AMF context
type GnbAmf struct {
	/* Indicates wether NGSetup was successful or not*/
	NgSetupStatus bool
	AmfIp         string
	AmfName       string
	AmfPort       uint16
	/* Relative AMF Capacity */
	RelCap          int64
	ServedGuamiList []models.Guami
	PlmnSupportList []factory.PlmnSupportItem
	/*Socket Connection*/
	Conn net.Conn
}

func NewGnbAmf(ip string, port uint16) *GnbAmf {
	return &GnbAmf{
		AmfIp:           ip,
		AmfPort:         port,
		ServedGuamiList: NewServedGUAMIList(),
		PlmnSupportList: NewPlmnSupportList(),
	}
}

func (amf *GnbAmf) GetIpAddr() string {
	return amf.AmfIp
}

func (amf *GnbAmf) GetPort() uint16 {
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
	return make([]models.Guami, 0, context.MaxNumOfServedGuamiList)
}

func NewPlmnSupportList() []factory.PlmnSupportItem {
	return make([]factory.PlmnSupportItem, 0, context.MaxNumOfPLMNs)
}
