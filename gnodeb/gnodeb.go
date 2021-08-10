// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"fmt"
	"gnbsim/util/test"

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
	Tac     []byte
	/* Default AMF to connect to */
	DefaultAmf *GnbAmf

	/* Control Plane transport */
	CpTransport transport
}

// Init initializes the GNodeB struct var and connects to the default AMF
func (gnb *GNodeB) Init() error {
	fmt.Printf("Default gNodeB configuration : %#v\n", *gnb)
	gnb.CpTransport = &gnbCTransport{gnb}

	if gnb.DefaultAmf == nil {
		fmt.Println("Default AMF not configured, continuing ...")
		return nil
	}

	err := gnb.ConnectToAmf(gnb.DefaultAmf)
	if err != nil {
		fmt.Println("failed to connect to amf : ", err)
		return err
	}
	successfulOutcome, err := gnb.PerformNgSetup(gnb.DefaultAmf)
	if !successfulOutcome || err != nil {
		fmt.Println("failed to perform NG Setup procedure : ", err)
		return err
	}

	go gnb.CpTransport.ReceiveFromPeer(gnb.DefaultAmf)

	return nil
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
	return
}

// PerformNGSetup sends the NGSetupRequest to the provided GnbAmf.
// It waits for the response, process the response and informs whether it was
// SuccessfulOutcome or UnsuccessfulOutcome
func (gnb *GNodeB) PerformNgSetup(amf *GnbAmf) (status bool, err error) {

	// Forming NGSetupRequest with configured parameters
	ngSetupReq, err := test.GetNGSetupRequest(gnb.Tac, gnb.GnbId, 24, gnb.GnbName)
	if err != nil {
		fmt.Println("failed to create setupRequest message")
		return
	}

	// Sending NGSetupRequest to AMF
	ngSetupResp, err := gnb.CpTransport.SendToPeerBlock(amf, ngSetupReq)
	if err != nil {
		fmt.Println("failed to send NGSetupRequest to AMF, error:", err)
		return
	}
	err = gnb.dispatch(amf, ngSetupResp)
	if err != nil {
		fmt.Println("Unexpected erro in NGSetupResponse")
		return
	}

	return amf.GetNgSetupStatus(), nil
}

// dispatch decodes an incoming NGAP message and routes it to the corresponding
// handlers or a GnbCpUe
func (gnb *GNodeB) dispatch(amf *GnbAmf, pkt []byte) error {
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
			HandleNgSetupResponse(amf, pdu)
		}
	case ngapType.NGAPPDUPresentUnsuccessfulOutcome:
		unsuccessfulOutcome := pdu.UnsuccessfulOutcome
		if unsuccessfulOutcome == nil {
			return fmt.Errorf("UnSuccessful Outcome is nil")
		}
		switch unsuccessfulOutcome.ProcedureCode.Value {
		case ngapType.ProcedureCodeNGSetup:
			HandleNgSetupFailure(amf, pdu)
		}
	}

	return nil
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}
