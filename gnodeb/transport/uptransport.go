// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"gnbsim/logger"
	"gnbsim/transportcommon"
	"net"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Need to check if NGAP may exceed this limit
var MAX_UDP_PKT_LEN int = 65507

//TODO: Should have a context variable which when cancelled will result in
// the termination of the ReceiveFromPeer handler

// GnbUpTransport represents the User Plane transport of the GNodeB
type GnbUpTransport struct {
	GnbInstance *context.GNodeB

	/* UDP Connection without any association with peers */
	Conn *net.UDPConn

	/* logger */
	Log *logrus.Entry
}

func NewGnbUpTransport(gnb *context.GNodeB) *GnbUpTransport {
	transport := &GnbUpTransport{}
	transport.GnbInstance = gnb
	transport.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "UserPlaneTransport"})

	return transport
}

func (upTprt *GnbUpTransport) Init() error {
	gnb := upTprt.GnbInstance
	ipPort := net.JoinHostPort(gnb.GnbN3Ip, strconv.Itoa(gnb.GnbN3Port))
	addr, err := net.ResolveUDPAddr("udp", ipPort)
	if err != nil {
		upTprt.Log.Errorln("ResolveUDPAddr returned:", err)
		return fmt.Errorf("invalid ip or port: %v", ipPort)
	}

	upTprt.Conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		upTprt.Log.Errorln("ListenUDP returned:", err)
		return fmt.Errorf("failed to create udp socket: %v", ipPort)
	}

	go upTprt.ReceiveFromPeer(nil)

	upTprt.Log.Infoln("User Plane transport listening on:", ipPort)
	return nil
}

// SendToPeer sends a GTP-U encoded packet to the specified UPF over the socket
func (upTprt *GnbUpTransport) SendToPeer(peer transportcommon.TransportPeer,
	pkt []byte) (err error) {

	err = upTprt.CheckTransportParam(peer, pkt)
	if err != nil {
		return err
	}

	upf := peer.(*context.GnbUpf)

	pktLen := len(pkt)
	n, err := upTprt.Conn.WriteTo(pkt, upf.UpfAddr)
	if err != nil {
		upTprt.Log.Errorln("WriteTo returned:", err)
		return fmt.Errorf("failed to write on socket")
	} else if n != pktLen {
		return fmt.Errorf("total bytes:%v, written bytes:%v", pktLen, n)
	} else {
		upTprt.Log.Infof("Sent UDP Packet, length: %v bytes\n", n)
	}

	return
}

// ReceiveFromPeer continuously waits for an incoming message from the UPF
// It then routes the message to the GnbUpfWorker
func (upTprt *GnbUpTransport) ReceiveFromPeer(peer transportcommon.TransportPeer) {
	for {
		recvMsg := make([]byte, MAX_UDP_PKT_LEN)
		//TODO Handle notification, info
		n, srcAddr, err := upTprt.Conn.ReadFromUDP(recvMsg)
		if err != nil {
			upTprt.Log.Errorln("ReadFromUDP returned:", err)
		}
		srcIp := srcAddr.IP.String()
		upTprt.Log.Infof("Read %v bytes from %v:%v\n", n, srcIp, srcAddr.Port)

		gnbupf := upTprt.GnbInstance.GnbPeers.GetGnbUpf(srcIp)
		if gnbupf == nil {
			upTprt.Log.Errorln("No UPF Context found corresponding to IP:", srcIp)
			continue
		}
		tMsg := &common.TransportMessage{}
		tMsg.RawPkt = recvMsg[:n]
		gnbupf.ReadChan <- tMsg
		upTprt.Log.Traceln("Forwarded UDP packet to UPF Worker")
	}
}

func (upTprt *GnbUpTransport) CheckTransportParam(peer transportcommon.TransportPeer,
	pkt []byte) error {

	upf := peer.(*context.GnbUpf)

	if upf == nil {
		return fmt.Errorf("UPF is nil")
	}

	if len(pkt) == 0 {
		return fmt.Errorf("packet len is 0")
	}

	if upf.UpfAddr == nil {
		return fmt.Errorf("UPF address is nil")
	}

	return nil
}

func (upTprt *GnbUpTransport) SendToPeerBlock(peer transportcommon.TransportPeer, pkt []byte) ([]byte, error) {
	return nil, nil
}

func (upTprt *GnbUpTransport) ConnectToPeer(peer transportcommon.TransportPeer) error {
	return nil
}
