// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"net"
	"strconv"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/sirupsen/logrus"
)

const GTP_U_PORT int = 2152

// GnbUpf holds the UPF context
type GnbUpf struct {
	UpfAddr  *net.UDPAddr
	GnbUpUes *GnbUeDao
	Log      *logrus.Entry

	// GnbUpf Reads messages from transport, GnbUpUe and GNodeB
	ReadChan chan common.InterfaceMessage

	UpfIpString string
}

func NewGnbUpf(ip string) *GnbUpf {
	gnbupf := &GnbUpf{}

	gnbupf.Log = logger.GNodeBLog.WithFields(logrus.Fields{
		"subcategory":  "GnbUpf",
		logger.FieldIp: ip,
	})

	ipPort := net.JoinHostPort(ip, strconv.Itoa(GTP_U_PORT))
	addr, err := net.ResolveUDPAddr("udp", ipPort)
	if err != nil {
		gnbupf.Log.Errorln("ResolveUDPAddr returned:", err)
		return nil
	}

	gnbupf.ReadChan = make(chan common.InterfaceMessage, 10)
	gnbupf.GnbUpUes = NewGnbUeDao()
	gnbupf.UpfAddr = addr
	gnbupf.UpfIpString = addr.IP.String()

	return gnbupf
}

func (upf *GnbUpf) GetIpAddr() string {
	return upf.UpfIpString
}

func (upf *GnbUpf) GetPort() int {
	return upf.UpfAddr.Port
}
