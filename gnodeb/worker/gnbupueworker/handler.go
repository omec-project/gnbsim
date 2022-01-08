// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbupueworker

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"gnbsim/util/test"
)

func HandleUlMessage(gnbue *context.GnbUpUe, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Traceln("Handling UL Packet from UE")

	if msg.GetEventType() == common.END_MARKER_EVENT {
		gnbue.Log.Debugln("Received last uplink data packet")
		gnbue.EndMarkerRecvd = true
		return nil
	}

	userDataMsg := msg.(*common.UserDataMessage)
	encodedMsg, err := test.BuildGpduMessage(userDataMsg.Payload, gnbue.UlTeid)
	if err != nil {
		gnbue.Log.Errorln("BuildGpduMessage() returned:", err)
		return fmt.Errorf("failed to encode gpdu")
	}
	err = gnbue.Gnb.UpTransport.SendToPeer(gnbue.Upf, encodedMsg)
	if err != nil {
		gnbue.Log.Errorln("UP Transport SendToPeer() returned:", err)
		return fmt.Errorf("failed to send gpdu")
	}
	gnbue.Log.Traceln("Sent UL Packet from UE to UPF")
	return nil
}

func HandleDlMessage(gnbue *context.GnbUpUe, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Traceln("Handling DL Packet from UPF Worker")

	userDataMsg := msg.(*common.UserDataMessage)

	if len(userDataMsg.Payload) == 0 {
		return fmt.Errorf("empty t-pdu")
	}

	/* TODO: Parse QFI and check if it exists in GnbUpUe. In real world,
	   gNb may use the QFI to find a corresponding DRB
	*/

	userDataMsg.Event = common.DL_UE_DATA_TRANSFER_EVENT
	gnbue.WriteUeChan <- userDataMsg
	gnbue.Log.Infoln("Sent DL user data packet to UE")

	return nil
}

func HandleQuitEvent(gnbue *context.GnbUpUe, intfcMsg common.InterfaceMessage) (err error) {
	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.END_MARKER_EVENT
	gnbue.WriteUeChan <- userDataMsg
	gnbue.WriteUeChan = nil

	// Drain all the messages until END MARKER is received.
	// This ensures that the transmitting go routine is not blocked while
	// sending data on this channel
	if gnbue.EndMarkerRecvd != true {
		for pkt := range gnbue.ReadUlChan {
			if pkt.GetEventType() == common.END_MARKER_EVENT {
				gnbue.Log.Debugln("Received last uplink data packet")
				break
			}
		}
	}

	gnbue.Gnb.DlTeidGenerator.FreeID(int64(gnbue.DlTeid))
	gnbue.Log.Infoln("Gnb User-plane UE Context terminated")

	return nil
}
