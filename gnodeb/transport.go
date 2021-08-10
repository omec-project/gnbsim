// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"fmt"
	"io"
	"syscall"

	"git.cs.nctu.edu.tw/calee/sctp"
)

// Need to check if NGAP may exceed this limit
var MAX_PKT_LEN int = 2048

type transport interface {
	SendToPeerBlock(amf *GnbAmf, pkt []byte) ([]byte, error)
	SendToPeer(amf *GnbAmf, pkt []byte) (err error)
	ReceiveFromPeer(amf *GnbAmf)
}

//TODO: Should have a context variable which when cancelled will result in
// the termination of the ReceiveFromPeer handler

// gnbCTransport represents the control plane transport of the GNodeB
type gnbCTransport struct {
	gnbInstance *GNodeB
}

//TODO Should add timeout

// SendToPeer sends an NGAP encoded packet to the specified AMF over the socket
// connection and waits for the response
func (cpTprt *gnbCTransport) SendToPeerBlock(amf *GnbAmf, pkt []byte) ([]byte, error) {
	err := cpTprt.SendToPeer(amf, pkt)
	if err != nil {
		fmt.Println("SendToPeer failed with err:", err)
		return nil, err
	}

	recvMsg := make([]byte, MAX_PKT_LEN)
	conn := amf.Conn.(*sctp.SCTPConn)
	n, _, _, err := conn.SCTPRead(recvMsg)
	if err != nil {
		fmt.Println("SCTPRead failed due to error:", err)
		return nil, err
	}

	fmt.Printf("Read %v bytes from %v\n", n, conn.RemoteAddr())
	return recvMsg[:n], nil
}

// SendToPeer sends an NGAP encoded packet to the specified AMF over the socket
// connection
func (cpTprt *gnbCTransport) SendToPeer(amf *GnbAmf, pkt []byte) (err error) {
	fmt.Println("gnbcTransport :: sendToPeer called")

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
func (cpTprt *gnbCTransport) ReceiveFromPeer(amf *GnbAmf) {
	fmt.Println("gnbcTransport :: ReciveFromPeer called")

	defer func() {
		if err := amf.Conn.Close(); err != nil && err != syscall.EBADF {
			fmt.Println("close connection error: %+v", err)
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
				fmt.Println("Read EOF from client")
				return
			case syscall.EAGAIN:
				fmt.Println("SCTP read timeout")
				continue
			case syscall.EINTR:
				fmt.Println("SCTPRead: %+v", err)
				continue
			default:
				fmt.Println("Handle connection[addr: %+v] error: %+v", amf.Conn.RemoteAddr(), err)
				return
			}
		}

		fmt.Printf("Read %v bytes from %v\n", n, conn)
		cpTprt.gnbInstance.dispatch(amf, recvMsg[:n])
	}
}

func checkTransportParam(amf *GnbAmf, pkt []byte) error {

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
