// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	transport "gnbsim/transportcommon"

	"github.com/sirupsen/logrus"
)

// GNodeB holds the context for a gNodeB. It manages the control plane and
// user plane layer of a gNodeB.
type GNodeB struct {
	//TODO IP and port should be the property of transport var
	GnbIp   string
	GnbPort uint16
	GnbName string
	GnbId   []byte
	Tac     []byte
	GnbUes  *GnbUeDao

	/*channel to notify all the go routines corresponding to this GNodeB instance to stop*/
	Quit chan int

	/* Default AMF to connect to */
	DefaultAmf *GnbAmf

	/* Control Plane transport */
	CpTransport transport.Transport

	/* logger */
	Log *logrus.Entry
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}
