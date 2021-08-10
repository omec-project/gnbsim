// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"fmt"
	"gnbsim/util/test"
	"net"

	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
)

// GNodeB holds the context for a gNodeB. It manages the control plane and
// user plane layer of a gNodeB.
type GNodeB struct {
	//TODO IP and port should be the property of transport var
	GnbIp   string
	GnbPort uint16
	GnbName string
	GnbId   []byte
	/* Default AMF to connect to */
	DefaultAmf *GnbAmf
	Tac        string

	/* Control Plane transport */
	CpTransport transport
}

// Init initializes the GNodeB struct var and connects to the default AMF
func (gnb *GNodeB) Init() error {
	fmt.Printf("Default gNodeB configuration : %#v", *gnb)
	gnb.CpTransport = &gnbCTransport{gnb}
	err := gnb.ConnectToAmf(gnb.DefaultAmf)
	if err != nil {
		fmt.Println("failed to connect to amf : ", err)
	}
	return err
}

//TODO this should be in transport as ConnectToPeer

// ConnectToAmf establishes SCTP connection with the AMF and initiates NG Setup
// Procedure.
func (gnb *GNodeB) ConnectToAmf(amf *GnbAmf) (err error) {
	fmt.Println("gnodeb : ConnectToAmf called, AMF details :", *amf)
	amf.Conn, err = test.ConnectToAmf(amf.AmfIp, gnb.GnbIp, int(amf.AmfPort), int(gnb.GnbPort))
	if err != nil {
		fmt.Println("Failed to connect to AMF ", *amf)
		return
	}
	fmt.Println("Success - connected to AMF ", *amf)
	go gnb.CpTransport.ReceiveFromPeer(amf)
	err = gnb.SendNGSetupRequest(gnb.DefaultAmf)
	return
}

// SendNGSetupRequest forms and sends an NG Setup Request to the provided AMF
// instance
func (gnb *GNodeB) SendNGSetupRequest(amf *GnbAmf) error {
	fmt.Println("gnodeb: Sending NGSetup Request to Amf:", gnb.DefaultAmf)

	// Forming NGSetupRequest with configured parameters
	sendMsg, err := test.GetNGSetupRequest(gnb.GnbId, 24, gnb.GnbName)
	if err != nil {
		fmt.Println("failed to create setupRequest message")
		return err
	}

	// Sending NGSetupRequest to AMF
	err = gnb.CpTransport.SendToPeer(amf, sendMsg)
	if err != nil {
		fmt.Println("failed to send NGSetupRequest to AMF, error:", err)
	}

	return err
}

// dispatch decodes an incoming NGAP message and routes it to the corresponding
// handlers or a GnbCpUe
func (gnb *GNodeB) dispatch(conn net.Conn, pkt []byte) error {
	// decoding the incoming packet
	pdu, err := ngap.Decoder(pkt)
	if err != nil {
		return fmt.Errorf("NGAP decode error : %+v", err)
	}

	// routing to correct handlers
	switch pdu.Present {
	case ngapType.NGAPPDUPresentSuccessfulOutcome:
		successfulOutcome := pdu.SuccessfulOutcome
		if successfulOutcome == nil {
			return fmt.Errorf("successful Outcome is nil")
		}
		switch successfulOutcome.ProcedureCode.Value {
		case ngapType.ProcedureCodeNGSetup:
			//TODO : fetch amf from map based on the connection
			HandleNGSetupResponse(gnb.DefaultAmf, pdu)
		}
	}
	return nil
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}
