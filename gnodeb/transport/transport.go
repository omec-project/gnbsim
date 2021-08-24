// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package transport

import (
	"fmt"
	"gnbsim/gnodeb/context"
	"gnbsim/gnodeb/worker/gnbamfworker"
	"gnbsim/transportcommon"
	"io"
	"log"
	"syscall"

	"git.cs.nctu.edu.tw/calee/sctp"
)

// Need to check if NGAP may exceed this limit
var MAX_PKT_LEN int = 2048

//TODO: Should have a context variable which when cancelled will result in
// the termination of the ReceiveFromPeer handler

// gnbCTransport represents the control plane transport of the GNodeB
type GnbCTransport struct {
	GnbInstance *context.GNodeB
}

//TODO Should add timeout

// SendToPeer sends an NGAP encoded packet to the specified AMF over the socket
// connection and waits for the response
func (cpTprt *GnbCTransport) SendToPeerBlock(peer transportcommon.TransportPeer, pkt []byte) ([]byte, error) {
	amf := peer.(*context.GnbAmf)
	err := cpTprt.SendToPeer(amf, pkt)
	if err != nil {
		log.Println("SendToPeer failed with err:", err)
		return nil, err
	}

	recvMsg := make([]byte, MAX_PKT_LEN)
	conn := amf.Conn.(*sctp.SCTPConn)
	n, _, _, err := conn.SCTPRead(recvMsg)
	if err != nil {
		log.Println("SCTPRead failed due to error:", err)
		return nil, err
	}

	fmt.Printf("Read %v bytes from %v\n", n, conn.RemoteAddr())
	return recvMsg[:n], nil
}

// SendToPeer sends an NGAP encoded packet to the specified AMF over the socket
// connection
func (cpTprt *GnbCTransport) SendToPeer(peer transportcommon.TransportPeer, pkt []byte) (err error) {
	amf := peer.(*context.GnbAmf)
	log.Println("gnbcTransport :: sendToPeer called")

	defer func() {
		recerr := recover()
		if recerr != nil {
			fmt.Printf("Recovered panic in SendToPeer, error: %+v\n", recerr)
			err = fmt.Errorf("SendToPeer() panic")
		}
	}()

	err = checkTransportParam(amf, pkt)
	if err != nil {
		return err
	}

	if n, err := amf.Conn.Write(pkt); err != nil || n != len(pkt) {
		return fmt.Errorf("failed to write message: %+v", err)
	} else {
		fmt.Printf("Wrote %v bytes\n", n)
	}

	return
}

// ReceiveFromPeer continuously waits for an incoming message from the AMF
// It then calls dispatch to route the message to the handlers/GnbCpUe
func (cpTprt *GnbCTransport) ReceiveFromPeer(peer transportcommon.TransportPeer) {
	amf := peer.(*context.GnbAmf)
	log.Println("gnbcTransport :: ReciveFromPeer called")

	defer func() {
		if err := amf.Conn.Close(); err != nil && err != syscall.EBADF {
			log.Println("close connection error:", err)
		}

	}()

	conn := amf.Conn.(*sctp.SCTPConn)
	for {
		recvMsg := make([]byte, MAX_PKT_LEN)
		//TODO Handle notification, info
		n, _, _, err := conn.SCTPRead(recvMsg)
		if err != nil {
			switch err {
			case io.EOF, io.ErrUnexpectedEOF:
				log.Println("Read EOF from client")
				return
			case syscall.EAGAIN:
				log.Println("SCTP read timeout")
				continue
			case syscall.EINTR:
				log.Printf("SCTPRead: %+v\n", err)
				continue
			default:
				log.Printf("Handle connection[addr: %+v] error: %+v\n", amf.Conn.RemoteAddr(), err)
				return
			}
		}

		fmt.Printf("Read %v bytes from %v\n", n, conn)
		//TODO Post to gnbamfworker channel
		gnbamfworker.HandleMessage(cpTprt.GnbInstance, amf, recvMsg[:n])
	}
}

func checkTransportParam(amf *context.GnbAmf, pkt []byte) error {

	if amf == nil {
		return fmt.Errorf("amf is nil")
	}

	if len(pkt) == 0 {
		return fmt.Errorf("packet len is 0")
	}

	if amf.Conn == nil {
		return fmt.Errorf("amf conn is nil")
	}

	if amf.Conn.RemoteAddr() == nil {
		return fmt.Errorf("ran addr is nil")
	}

	return nil
}
