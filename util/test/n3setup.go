// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"net"
)

func ConnectToUpf(enbIP, upfIP string, gnbPort, upfPort int) (*net.UDPConn, error) {
	upfAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", upfIP, upfPort))
	if err != nil {
		return nil, err
	}
	gnbAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", enbIP, gnbPort))
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", gnbAddr, upfAddr)
}
