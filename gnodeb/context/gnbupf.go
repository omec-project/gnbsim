// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"gnbsim/logger"
	"net"
	"strconv"

	"github.com/sirupsen/logrus"
)

const GTP_U_PORT int = 2152

// GnbUpf holds the UPF context
type GnbUpf struct {
	UpfAddr *net.UDPAddr

	/* logger */
	Log *logrus.Entry
}

func NewGnbUpf(ip string) *GnbUpf {
	gnbupf := &GnbUpf{}

	gnbupf.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbUpf",
		logger.FieldIp: ip})

	ipPort := net.JoinHostPort(ip, strconv.Itoa(GTP_U_PORT))
	addr, err := net.ResolveUDPAddr("udp", ipPort)
	if err != nil {
		gnbupf.Log.Errorln("ResolveUDPAddr returned:", err)
		return nil
	}

	gnbupf.UpfAddr = addr

	return gnbupf
}

func (upf *GnbUpf) GetIpAddr() string {
	return upf.UpfAddr.IP.String()
}

func (upf *GnbUpf) GetPort() int {
	return upf.UpfAddr.Port
}
