// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"fmt"
	"gnbsim/gnodeb/context"
	"gnbsim/gnodeb/transport"
	"gnbsim/gnodeb/worker/gnbamfworker"
	"gnbsim/gnodeb/worker/gnbueworker"
	"gnbsim/util/test"
	"log"

	intfc "gnbsim/interfacecommon"
)

// Init initializes the GNodeB struct var and connects to the default AMF
func Init(gnb *context.GNodeB) error {
	fmt.Printf("Default gNodeB configuration : %#v\n", *gnb)
	gnb.CpTransport = &transport.GnbCTransport{GnbInstance: gnb}
	gnb.GnbUes = &context.GnbUeDao{}

	if gnb.DefaultAmf == nil {
		log.Println("Default AMF not configured, continuing ...")
		return nil
	}

	err := ConnectToAmf(gnb, gnb.DefaultAmf)
	if err != nil {
		log.Println("failed to connect to amf : ", err)
		return err
	}
	successfulOutcome, err := PerformNgSetup(gnb, gnb.DefaultAmf)
	if !successfulOutcome || err != nil {
		log.Println("failed to perform NG Setup procedure : ", err)
		return err
	}

	go gnb.CpTransport.ReceiveFromPeer(gnb.DefaultAmf)

	return nil
}

func QuitGnb(gnb *context.GNodeB) {
	log.Println("Shutting Down GNodeB:", gnb.GnbName)
	close(gnb.Quit)
}

//TODO this should be in transport as ConnectToPeer

// ConnectToAmf establishes SCTP connection with the AMF and initiates NG Setup
// Procedure.
func ConnectToAmf(gnb *context.GNodeB, amf *context.GnbAmf) (err error) {
	log.Println("gnodeb : ConnectToAmf called, AMF details :", *amf)
	amf.Conn, err = test.ConnectToAmf(amf.AmfIp, gnb.GnbIp, int(amf.AmfPort), int(gnb.GnbPort))
	if err != nil {
		log.Println("Failed to connect to AMF ", *amf)
		return
	}
	log.Println("Success - connected to AMF ", *amf)
	return
}

// PerformNGSetup sends the NGSetupRequest to the provided GnbAmf.
// It waits for the response, process the response and informs whether it was
// SuccessfulOutcome or UnsuccessfulOutcome
func PerformNgSetup(gnb *context.GNodeB, amf *context.GnbAmf) (status bool, err error) {

	// Forming NGSetupRequest with configured parameters
	ngSetupReq, err := test.GetNGSetupRequest(gnb.Tac, gnb.GnbId, 24, gnb.GnbName)
	if err != nil {
		log.Println("failed to create setupRequest message")
		return
	}

	// Sending NGSetupRequest to AMF
	ngSetupResp, err := gnb.CpTransport.SendToPeerBlock(amf, ngSetupReq)
	if err != nil {
		log.Println("failed to send NGSetupRequest to AMF, error:", err)
		return
	}
	err = gnbamfworker.HandleMessage(gnb, amf, ngSetupResp)
	if err != nil {
		log.Println("Unexpected erro in NGSetupResponse")
		return
	}

	return amf.GetNgSetupStatus(), nil
}

// RequestConnection should be called by UE that is willing to connect to this GNodeB
func RequestConnection(gnb *context.GNodeB, uemsg *intfc.UuMessage) chan intfc.InterfaceMessage {
	// TODO Get NGAP Id from NGAP ID Pool
	gnbUe := gnb.GnbUes.GetGnbUe(1)
	if gnbUe != nil {
		fmt.Printf("Error: Cannot process Register Request. GnbUe context already exists.")
		return nil
	}
	gnbUe = context.NewGnbUe(1, gnb, gnb.DefaultAmf)
	gnb.GnbUes.AddGnbUe(1, gnbUe)

	go gnbueworker.Init(gnbUe)
	//Channel on which UE can write message to GnbUe and from which GnbUe will
	//be reading.
	ch := gnbUe.ReadChan
	ch <- uemsg
	return ch
}
