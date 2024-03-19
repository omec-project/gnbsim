// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbupueworker

import (
	"fmt"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/test"
)

func HandleUlMessage(gnbue *gnbctx.GnbUpUe, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Traceln("Handling UL Packet from UE")

	if msg.GetEventType() == common.LAST_DATA_PKT_EVENT {
		gnbue.Log.Debugln("Received last uplink data packet")
		gnbue.LastDataPktRecvd = true
		return nil
	}

	userDataMsg := msg.(*common.UserDataMessage)
	encodedMsg, err := test.BuildGpduMessage(userDataMsg.Payload, gnbue.UlTeid)
	if err != nil {
		gnbue.Log.Errorln("BuildGpduMessage() returned:", err)
		return fmt.Errorf("failed to encode gpdu")
	}
	err = gnbue.Gnb.UpTransport.SendToPeer(gnbue.Upf, encodedMsg, 0)
	if err != nil {
		gnbue.Log.Errorln("UP Transport SendToPeer() returned:", err)
		return fmt.Errorf("failed to send gpdu")
	}
	gnbue.Log.Traceln("Sent UL Packet from UE to UPF")
	return nil
}

func HandleDlMessage(gnbue *gnbctx.GnbUpUe, intfcMsg common.InterfaceMessage) (err error) {
	gnbue.Log.Traceln("Handling DL Packet from UPF Worker")

	msg := intfcMsg.(*common.N3Message)
	if len(msg.Pdu.Payload) == 0 {
		return fmt.Errorf("empty t-pdu")
	}

	ueDataMsg := &common.UserDataMessage{}
	ueDataMsg.Payload = msg.Pdu.Payload

	optHdr := msg.Pdu.OptHdr
	if optHdr != nil {
		if optHdr.NextHdrType == test.PDU_SESS_CONTAINER_EXT_HEADER_TYPE {
			// TODO: Write a generic function to process all the extension
			// headers and return a map(ext header type - ext headers)
			// and user data
			var extHdr *test.PduSessContainerExtHeader
			ueDataMsg.Payload, extHdr, err = test.DecodePduSessContainerExtHeader(msg.Pdu.Payload)
			if err != nil {
				return fmt.Errorf("failed to decode pdu session container extension header:%v", err)
			}
			ueDataMsg.Qfi = new(uint8)
			*ueDataMsg.Qfi = extHdr.Qfi
			gnbue.Log.Infoln("Received QFI value in downlink G-PDU:", extHdr.Qfi)
		}
	}

	ueDataMsg.Event = common.DL_UE_DATA_TRANSFER_EVENT
	gnbue.WriteUeChan <- ueDataMsg
	gnbue.Log.Infoln("Sent DL user data packet to UE")

	return nil
}

func HandleQuitEvent(gnbue *gnbctx.GnbUpUe, intfcMsg common.InterfaceMessage) (err error) {
	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.LAST_DATA_PKT_EVENT
	gnbue.WriteUeChan <- userDataMsg
	gnbue.WriteUeChan = nil

	// Drain all the messages until END MARKER is received.
	// This ensures that the transmitting go routine is not blocked while
	// sending data on this channel
	if !gnbue.LastDataPktRecvd {
		for pkt := range gnbue.ReadUlChan {
			if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
				gnbue.Log.Debugln("Received last uplink data packet")
				break
			}
		}
	}

	gnbue.Gnb.DlTeidGenerator.FreeID(int64(gnbue.DlTeid))
	gnbue.Log.Infoln("Gnb User-plane UE Context terminated")

	return nil
}
