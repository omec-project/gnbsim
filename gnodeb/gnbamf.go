// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"net"

	"github.com/free5gc/amf/context"
	"github.com/free5gc/amf/factory"
	"github.com/free5gc/openapi/models"
)

//TODO Shift this to context package within gnodeb

// GnbAmf holds the AMF context
type GnbAmf struct {
	AmfIp   string
	AmfName string
	AmfPort uint16
	/* Relative AMF Capacity */
	RelCap          int64
	ServedGuamiList []models.Guami
	PlmnSupportList []factory.PlmnSupportItem
	/*Socket Connection*/
	Conn net.Conn
}

func (amf *GnbAmf) SetAMFName(name string) {
	amf.AmfName = name
}

func (amf *GnbAmf) SetRelativeAMFCapacity(cap int64) {
	amf.RelCap = cap
}

func NewServedGUAMIList() []models.Guami {
	return make([]models.Guami, 0, context.MaxNumOfServedGuamiList)
}

func NewPlmnSupportList() []factory.PlmnSupportItem {
	return make([]factory.PlmnSupportItem, 0, context.MaxNumOfPLMNs)
}
