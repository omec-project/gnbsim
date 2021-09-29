// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"encoding/hex"
	"fmt"
	"gnbsim/common"
	"gnbsim/factory"
	"gnbsim/gnodeb/context"
	"gnbsim/gnodeb/transport"
	"gnbsim/gnodeb/worker/gnbamfworker"
	"gnbsim/gnodeb/worker/gnbueworker"
	"gnbsim/logger"
	"gnbsim/util/test"
	"log"
	"net"

	"github.com/free5gc/idgenerator"
)

func InitializeAllGnbs() error {
	gnbs := factory.AppConfig.Configuration.Gnbs
	for _, gnb := range gnbs {
		err := Init(gnb)
		if err != nil {
			gnb.Log.Errorln("Failed to initialize GNodeB, err:", err)
			return err
		}
	}
	return nil
}

// Init initializes the GNodeB struct var and connects to the default AMF
func Init(gnb *context.GNodeB) error {
	gnb.Log = logger.GNodeBLog.WithField(logger.FieldGnb, gnb.GnbName)
	gnb.Log.Traceln("Inititializing GNodeB")
	gnb.Log.Infoln("GNodeB IP:", gnb.GnbN2Ip, "GNodeB Port:", gnb.GnbN2Port)

	gnb.CpTransport = transport.NewGnbCpTransport(gnb)
	gnb.UpTransport = transport.NewGnbUpTransport(gnb)
	gnb.UpTransport.Init()
	gnb.GnbUes = &context.GnbUeDao{}
	gnb.RanUeNGAPIDGenerator = idgenerator.NewGenerator(1, context.MaxValueOfRanUeNgapId)

	if gnb.DefaultAmf == nil {
		gnb.Log.Infoln("Default AMF not configured, continuing ...")
		return nil
	}

	err := ConnectToAmf(gnb, gnb.DefaultAmf)
	if err != nil {
		gnb.Log.Errorln("ConnectToAmf returned:", err)
		return fmt.Errorf("failed to connect to amf")
	}

	successfulOutcome, err := PerformNgSetup(gnb, gnb.DefaultAmf)
	if !successfulOutcome || err != nil {
		gnb.Log.Errorln("PerformNgSetup returned:", err)
		return fmt.Errorf("failed to perform ng setup procedure")
	}

	go gnb.CpTransport.ReceiveFromPeer(gnb.DefaultAmf)

	gnb.Log.Traceln("GNodeB Initialized")
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

	if amf.AmfIp == "" {
		if amf.AmfHostName == "" {
			return fmt.Errorf("amf ip or host name not configured")
		}
		addrs, err := net.LookupHost(amf.AmfHostName)
		if err != nil {
			return fmt.Errorf("failed to resolve amf host name: %v, err: %v",
				amf.AmfHostName, err)
		}
		amf.AmfIp = addrs[0]
	}

	amf.Conn, err = test.ConnectToAmf(amf.AmfIp, gnb.GnbN2Ip, int(amf.AmfPort),
		int(gnb.GnbN2Port))
	if err != nil {
		return fmt.Errorf("failed to connect amf, ip: %v, port: %v, err: %v",
			amf.AmfIp, amf.AmfPort, err)
	}

	gnb.Log.Infoln("Connected to AMF, AMF IP:", amf.AmfIp, "AMF Port:", amf.AmfPort)
	return
}

// PerformNGSetup sends the NGSetupRequest to the provided GnbAmf.
// It waits for the response, process the response and informs whether it was
// SuccessfulOutcome or UnsuccessfulOutcome
func PerformNgSetup(gnb *context.GNodeB, amf *context.GnbAmf) (bool, error) {
	gnb.Log.Traceln("Performing NG Setup Procedure")

	var status bool
	tac, err := hex.DecodeString(gnb.Tac)
	if err != nil {
		gnb.Log.Errorln("DecodeString returned:", err)
		return status, fmt.Errorf("invalid TAC")
	}

	gnbId, err := hex.DecodeString(gnb.GnbId)
	if err != nil {
		gnb.Log.Errorln("DecodeString returned:", err)
		return status, fmt.Errorf("invalid gNB ID")
	}

	// Forming NGSetupRequest
	ngSetupReq, err := test.GetNGSetupRequest(tac, gnbId, 24, gnb.GnbName)
	if err != nil {
		gnb.Log.Errorln("GetNGSetupRequest returned:", err)
		return status, fmt.Errorf("failed to create ng setup request")
	}

	gnb.Log.Traceln("Sending NG Setup Request")
	ngSetupResp, err := gnb.CpTransport.SendToPeerBlock(amf, ngSetupReq)
	if err != nil {
		gnb.Log.Errorln("SendToPeerBlock returned:", err)
		return status, fmt.Errorf("failed to send ng setup request")
	}
	gnb.Log.Traceln("Received NG Setup Response")
	err = gnbamfworker.HandleMessage(gnb, amf, ngSetupResp)
	if err != nil {
		gnb.Log.Errorln("HandleMessage returned:", err)
		return status, fmt.Errorf("failed to handle ng setup response")
	}

	status = amf.GetNgSetupStatus()
	gnb.Log.Infoln("NG Setup Successful:", status)
	return status, nil
}

// RequestConnection should be called by UE that is willing to connect to this GNodeB
func RequestConnection(gnb *context.GNodeB, uemsg *common.UuMessage) (chan common.InterfaceMessage, error) {
	//TODO : Should have a map of supi and with the help of it check if same
	// SimUe sent a connection request
	ranUeNgapID, err := gnb.AllocateRanUeNgapID()
	if err != nil {
		gnb.Log.Errorln("AllocateRanUeNgapID returned:", err)
		return nil, fmt.Errorf("failed to allocate ran ue ngap id")
	}

	gnbUe := gnb.GnbUes.GetGnbUe(ranUeNgapID)
	if gnbUe != nil {
		return nil, fmt.Errorf("gnb ue context already exists")
	}

	gnbUe = context.NewGnbUe(ranUeNgapID, gnb, gnb.DefaultAmf)
	gnb.GnbUes.AddGnbUe(1, gnbUe)

	go gnbueworker.Init(gnbUe)
	//Channel on which UE can write message to GnbUe and from which GnbUe will
	//be reading.
	ch := gnbUe.ReadChan
	ch <- uemsg
	return ch, nil
}
