// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package transport

import (
	"fmt"
	"gnbsim/gnodeb/context"
	"gnbsim/gnodeb/worker/gnbamfworker"
	"gnbsim/logger"
	"gnbsim/transportcommon"
	"io"
	"syscall"

	"git.cs.nctu.edu.tw/calee/sctp"
	"github.com/sirupsen/logrus"
)

// Need to check if NGAP may exceed this limit
var MAX_SCTP_PKT_LEN int = 2048

//TODO: Should have a context variable which when cancelled will result in
// the termination of the ReceiveFromPeer handler

// GnbCpTransport represents the control plane transport of the GNodeB
type GnbCpTransport struct {
	GnbInstance *context.GNodeB

	/* logger */
	Log *logrus.Entry
}

func NewGnbCpTransport(gnb *context.GNodeB) *GnbCpTransport {
	transport := &GnbCpTransport{}
	transport.GnbInstance = gnb
	transport.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "ControlPlaneTransport"})

	return transport
}

//TODO Should add timeout

// SendToPeer sends an NGAP encoded packet to the specified AMF over the socket
// connection and waits for the response
func (cpTprt *GnbCpTransport) SendToPeerBlock(peer transportcommon.TransportPeer,
	pkt []byte) ([]byte, error) {

	err := cpTprt.SendToPeer(peer, pkt)
	if err != nil {
		cpTprt.Log.Errorln("SendToPeer returned err:", err)
		return nil, fmt.Errorf("failed to send packet")
	}

	amf := peer.(*context.GnbAmf)

	recvMsg := make([]byte, MAX_SCTP_PKT_LEN)
	conn := amf.Conn.(*sctp.SCTPConn)

	n, _, _, err := conn.SCTPRead(recvMsg)
	if err != nil {
		cpTprt.Log.Errorln("SCTPRead returned :", err)
		return nil, fmt.Errorf("failed to read from socket")
	}

	cpTprt.Log.Infof("Read %v bytes from %v\n", n, conn.RemoteAddr())
	return recvMsg[:n], nil
}

// SendToPeer sends an NGAP encoded packet to the specified AMF over the socket
// connection
func (cpTprt *GnbCpTransport) SendToPeer(peer transportcommon.TransportPeer,
	pkt []byte) (err error) {

	err = cpTprt.CheckTransportParam(peer, pkt)
	if err != nil {
		return err
	}

	amf := peer.(*context.GnbAmf)

	defer func() {
		recerr := recover()
		if recerr != nil {
			cpTprt.Log.Errorln("Recovered panic in SendToPeer, error:", recerr)
			err = fmt.Errorf("recovered panic")
		}
	}()

	if n, err := amf.Conn.Write(pkt); err != nil || n != len(pkt) {
		cpTprt.Log.Errorln("Write returned:", err)
		return fmt.Errorf("failed to write on socket")
	} else {
		cpTprt.Log.Infof("Wrote %v bytes\n", n)
	}

	return
}

// ReceiveFromPeer continuously waits for an incoming message from the AMF
// It then routes the message to the GnbAmfWorker
func (cpTprt *GnbCpTransport) ReceiveFromPeer(peer transportcommon.TransportPeer) {
	amf := peer.(*context.GnbAmf)

	defer func() {
		if err := amf.Conn.Close(); err != nil && err != syscall.EBADF {
			cpTprt.Log.Errorln("Close returned:", err)
		}

	}()

	conn := amf.Conn.(*sctp.SCTPConn)
	for {
		recvMsg := make([]byte, MAX_SCTP_PKT_LEN)
		//TODO Handle notification, info
		n, _, _, err := conn.SCTPRead(recvMsg)
		if err != nil {
			switch err {
			case io.EOF, io.ErrUnexpectedEOF:
				cpTprt.Log.Errorln("Read EOF from client")
				return
			case syscall.EAGAIN:
				cpTprt.Log.Warnln("SCTP read timeout")
				continue
			case syscall.EINTR:
				cpTprt.Log.Warnln("SCTPRead: %+v\n", err)
				continue
			default:
				cpTprt.Log.Errorln("Handle connection[addr: %+v] error: %+v\n", amf.Conn.RemoteAddr(), err)
				return
			}
		}

		cpTprt.Log.Infof("Read %v bytes from %v\n", n, conn)
		//TODO Post to gnbamfworker channel
		gnbamfworker.HandleMessage(cpTprt.GnbInstance, amf, recvMsg[:n])
	}
}

func (cpTprt *GnbCpTransport) CheckTransportParam(peer transportcommon.TransportPeer, pkt []byte) error {
	amf := peer.(*context.GnbAmf)

	if amf == nil {
		return fmt.Errorf("AMF is nil")
	}

	if len(pkt) == 0 {
		return fmt.Errorf("packet len is 0")
	}

	if amf.Conn == nil {
		return fmt.Errorf("AMF conn is nil")
	}

	if amf.Conn.RemoteAddr() == nil {
		return fmt.Errorf("AMF IP address is nil")
	}

	return nil
}

func (cpTprt *GnbCpTransport) Init() {
}
