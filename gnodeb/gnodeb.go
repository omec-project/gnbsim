// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnodeb

import (
	"fmt"
	"log"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/gnodeb/idrange"
	"github.com/omec-project/gnbsim/gnodeb/ngap"
	"github.com/omec-project/gnbsim/gnodeb/transport"
	"github.com/omec-project/gnbsim/gnodeb/worker/gnbamfworker"
	"github.com/omec-project/gnbsim/gnodeb/worker/gnbcpueworker"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/util/idgenerator"
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
func Init(gnb *gnbctx.GNodeB) error {
	gnb.Log = logger.GNodeBLog.WithField(logger.FieldGnb, gnb.GnbName)
	gnb.Log.Traceln("Inititializing GNodeB")
	gnb.Log.Infoln("GNodeB IP:", gnb.GnbN2Ip, "GNodeB Port:", gnb.GnbN2Port)

	gnb.CpTransport = transport.NewGnbCpTransport(gnb)
	gnb.UpTransport = transport.NewGnbUpTransport(gnb)
	err := gnb.UpTransport.Init()
	if err != nil {
		gnb.Log.Errorln("GnbUpTransport.Init returned", err)
		return fmt.Errorf("failed to initialize user plane transport")
	}
	gnb.GnbUes = gnbctx.NewGnbUeDao()
	gnb.GnbPeers = gnbctx.NewGnbPeerDao()
	start, end := idrange.GetIdRange()
	gnb.RanUeNGAPIDGenerator = idgenerator.NewGenerator(int64(start), int64(end))
	gnb.DlTeidGenerator = idgenerator.NewGenerator(int64(start), int64(end))

	if gnb.DefaultAmf == nil {
		gnb.Log.Infoln("Default AMF not configured, continuing ...")
		return nil
	}

	gnb.DefaultAmf.Init()

	err = gnb.CpTransport.ConnectToPeer(gnb.DefaultAmf)
	if err != nil {
		gnb.Log.Errorln("ConnectToPeer returned:", err)
		return fmt.Errorf("failed to connect to amf")
	}

	successfulOutcome, err := PerformNgSetup(gnb, gnb.DefaultAmf)
	if !successfulOutcome || err != nil {
		gnb.Log.Errorln("PerformNgSetup returned:", err)
		return fmt.Errorf("failed to perform ng setup procedure")
	}

	go gnb.CpTransport.ReceiveFromPeer(gnb.DefaultAmf)

	gnb.Log.Tracef("GNodeB Initialized %v ", gnb)
	return nil
}

func QuitGnb(gnb *gnbctx.GNodeB) {
	log.Println("Shutting Down GNodeB:", gnb.GnbName)
	close(gnb.Quit)
}

// PerformNGSetup sends the NGSetupRequest to the provided GnbAmf.
// It waits for the response, process the response and informs whether it was
// SuccessfulOutcome or UnsuccessfulOutcome
func PerformNgSetup(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf) (bool, error) {
	gnb.Log.Traceln("Performing NG Setup Procedure")

	var status bool

	// Forming NGSetupRequest
	ngSetupReq, err := ngap.GetNGSetupRequest(gnb)
	if err != nil {
		gnb.Log.Errorln("GetNGSetupRequest returned:", err)
		return status, fmt.Errorf("failed to create ng setup request")
	}

	gnb.Log.Traceln("Sending NG Setup Request")
	ngSetupResp, err := gnb.CpTransport.SendToPeerBlock(amf, ngSetupReq, 0)
	if err != nil {
		gnb.Log.Errorln("SendToPeerBlock returned:", err)
		return status, fmt.Errorf("failed to send ng setup request")
	}
	gnb.Log.Traceln("Received NG Setup Response")
	err = gnbamfworker.HandleMessage(gnb, amf, ngSetupResp, 0)
	if err != nil {
		gnb.Log.Errorln("HandleMessage returned:", err)
		return status, fmt.Errorf("failed to handle ng setup response")
	}

	status = amf.GetNgSetupStatus()
	gnb.Log.Infoln("NG Setup Successful:", status)
	return status, nil
}

// RequestConnection should be called by UE that is willing to connect to this GNodeB
func RequestConnection(gnb *gnbctx.GNodeB, uemsg *common.UuMessage) (chan common.InterfaceMessage, error) {
	ranUeNgapID, err := gnb.AllocateRanUeNgapID()
	if err != nil {
		gnb.Log.Errorln("AllocateRanUeNgapID returned:", err)
		return nil, fmt.Errorf("failed to allocate ran ue ngap id")
	}

	gnbUe := gnbctx.NewGnbCpUe(ranUeNgapID, gnb, gnb.DefaultAmf)
	gnb.GnbUes.AddGnbCpUe(ranUeNgapID, gnbUe)

	// TODO: Launching a GO Routine for gNB and handling the waitgroup
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		gnbcpueworker.Init(gnbUe)
	}()
	// Channel on which UE can write message to GnbUe and from which GnbUe will
	// be reading.
	ch := gnbUe.ReadChan
	ch <- uemsg
	return ch, nil
}
