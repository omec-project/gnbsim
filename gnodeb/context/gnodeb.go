// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	transport "gnbsim/transportcommon"

	"github.com/free5gc/idgenerator"
	"github.com/sirupsen/logrus"
)

// GNodeB holds the context for a gNodeB. It manages the control plane and
// user plane layer of a gNodeB.
type GNodeB struct {
	//TODO IP and port should be the property of transport var
	GnbN2Ip              string `yaml:"n2IpAddr"`
	GnbN2Port            int    `yaml:"n2Port"`
	GnbN3Ip              string `yaml:"n3IpAddr"`
	GnbN3Port            int    `yaml:"n3Port"`
	GnbName              string `yaml:"name"`
	GnbId                string `yaml:"gnbId"`
	Tac                  string `yaml:"tac"`
	GnbUes               *GnbUeDao
	RanUeNGAPIDGenerator *idgenerator.IDGenerator

	/*channel to notify all the go routines corresponding to this GNodeB instance to stop*/
	Quit chan int

	/* Default AMF to connect to */
	DefaultAmf *GnbAmf `yaml:"defaultAmf"`

	/* Control Plane transport */
	CpTransport transport.Transport

	/* User Plane transport */
	UpTransport transport.Transport

	/* logger */
	Log *logrus.Entry
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}

func (gnb *GNodeB) AllocateRanUeNgapID() (int64, error) {
	return gnb.RanUeNGAPIDGenerator.Allocate()
}
