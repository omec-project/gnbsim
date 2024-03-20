// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	transport "github.com/omec-project/gnbsim/transportcommon"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/idgenerator"
	"github.com/sirupsen/logrus"
)

// GNodeB holds the context for a gNodeB. It manages the control plane and
// user plane layer of a gNodeB.
type GNodeB struct {
	// TODO IP and port should be the property of transport var
	GnbN2Ip              string                 `yaml:"n2IpAddr"`
	GnbN3Ip              string                 `yaml:"n3IpAddr"`
	GnbName              string                 `yaml:"name"`
	RanId                models.GlobalRanNodeId `yaml:"globalRanId"`
	GnbUes               *GnbUeDao
	GnbPeers             *GnbPeerDao
	RanUeNGAPIDGenerator *idgenerator.IDGenerator
	DlTeidGenerator      *idgenerator.IDGenerator
	Log                  *logrus.Entry

	/*channel to notify all the go routines corresponding to this GNodeB instance to stop*/
	Quit chan int

	/* Default AMF to connect to */
	DefaultAmf *GnbAmf `yaml:"defaultAmf"`

	/* Control Plane transport */
	CpTransport transport.Transport

	/* User Plane transport */
	UpTransport transport.Transport

	SupportedTaList []SupportedTA `yaml:"supportedTaList"`
	GnbN2Port       int           `yaml:"n2Port"`
	GnbN3Port       int           `yaml:"n3Port"`
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}

func (gnb *GNodeB) AllocateRanUeNgapID() (int64, error) {
	return gnb.RanUeNGAPIDGenerator.Allocate()
}

type SupportedTA struct {
	Tac               string              `yaml:"tac"`
	BroadcastPLMNList []BroadcastPLMNItem `yaml:"broadcastPlmnList"`
}

type BroadcastPLMNItem struct {
	PlmnId              models.PlmnId   `yaml:"plmnId"`
	TaiSliceSupportList []models.Snssai `yaml:"taiSliceSupportList"`
}
