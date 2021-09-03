// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"gnbsim/gnodeb/transport"
	"gnbsim/gnodeb/worker/gnbamfworker"
	"gnbsim/gnodeb/worker/gnbueworker"
	"gnbsim/util/test"
	"log"
)

// Init initializes the GNodeB struct var and connects to the default AMF
func Init(gnb *context.GNodeB) error {
	gnb.Log.Traceln("Inititializing GNodeB")
	gnb.Log.Infoln("GNodeB IP:", gnb.GnbIp, "GNodeB Port:", gnb.GnbPort)
	gnb.CpTransport = &transport.GnbCTransport{GnbInstance: gnb}
	gnb.GnbUes = &context.GnbUeDao{}

	if gnb.DefaultAmf == nil {
		gnb.Log.Traceln("Default AMF not configured, continuing ...")
		return nil
	}

	err := ConnectToAmf(gnb, gnb.DefaultAmf)
	if err != nil {
		gnb.Log.Errorln("failed to connect to amf : ", err)
		return err
	}
	successfulOutcome, err := PerformNgSetup(gnb, gnb.DefaultAmf)
	if !successfulOutcome || err != nil {
		gnb.Log.Errorln("failed to perform ng setup procedure : ", err)
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
	gnb.Log.Traceln("Connecting to AMF")
	amf.Conn, err = test.ConnectToAmf(amf.AmfIp, gnb.GnbIp, int(amf.AmfPort),
		int(gnb.GnbPort))
	if err != nil {
		gnb.Log.Errorln("failed to connect to AMF, AMF IP:", amf.AmfIp, "Error:", err)
		return
	}
	gnb.Log.Infoln("Connected to AMF, AMF IP:", amf.AmfIp, "AMF Port:", amf.AmfPort)
	return
}

// PerformNGSetup sends the NGSetupRequest to the provided GnbAmf.
// It waits for the response, process the response and informs whether it was
// SuccessfulOutcome or UnsuccessfulOutcome
func PerformNgSetup(gnb *context.GNodeB, amf *context.GnbAmf) (status bool, err error) {
	gnb.Log.Traceln("Performing NG Setup Procedure")

	// Forming NGSetupRequest with configured parameters
	ngSetupReq, err := test.GetNGSetupRequest(gnb.Tac, gnb.GnbId, 24, gnb.GnbName)
	if err != nil {
		gnb.Log.Errorln("failed to create setupRequest message")
		return
	}

	// Sending NGSetupRequest to AMF
	gnb.Log.Traceln("Sending NG Setup Request")
	ngSetupResp, err := gnb.CpTransport.SendToPeerBlock(amf, ngSetupReq)
	if err != nil {
		gnb.Log.Errorln("SendToPeerBlock Failed:", err)
		return
	}
	gnb.Log.Traceln("Received NG Setup Response")
	err = gnbamfworker.HandleMessage(gnb, amf, ngSetupResp)
	if err != nil {
		gnb.Log.Errorln("HandleMessage Failed:", err)
		return
	}

	return amf.GetNgSetupStatus(), nil
}

// RequestConnection should be called by UE that is willing to connect to this GNodeB
func RequestConnection(gnb *context.GNodeB, uemsg *common.UuMessage) chan common.InterfaceMessage {
	// TODO Get NGAP Id from NGAP ID Pool
	gnbUe := gnb.GnbUes.GetGnbUe(1)
	if gnbUe != nil {
		gnb.Log.Errorln("GnbUe context already exists")
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
