// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbupueworker

import (
	"context"
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/test"
)

func HandleUlMessage(gnbue *gnbctx.GnbUpUe, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Debugln("handling UL Packet from UE")

	if msg.GetEventType() == common.LAST_DATA_PKT_EVENT {
		gnbue.Log.Debugln("Received last uplink data packet")
		gnbue.LastDataPktRecvd = true
		return nil
	}

	// A session with no admitted QoS flow has nothing to stamp the packet with. Sending it
	// anyway marks it QFI 0, which is a QFI the UPF has no rule for once the network has
	// withdrawn the session's flows -- the packet is lost either way, but the gNB claiming a
	// flow it does not have hides why.
	qfi, ok := gnbue.AnyQfi()
	if !ok {
		return fmt.Errorf("no QoS flow on PDU session %v: dropping the uplink packet",
			gnbue.PduSessId)
	}

	userDataMsg := msg.(*common.UserDataMessage)
	encodedMsg, err := test.BuildGpduMessage(userDataMsg.Payload, gnbue.UlTeid, uint8(qfi))
	if err != nil {
		gnbue.Log.Errorln("BuildGpduMessage() returned:", err)
		return fmt.Errorf("failed to encode gpdu")
	}
	err = gnbue.Gnb.UpTransport.SendToPeer(gnbue.Upf, encodedMsg, 0)
	if err != nil {
		gnbue.Log.Errorln("UP Transport SendToPeer() returned:", err)
		return fmt.Errorf("failed to send gpdu")
	}
	gnbue.Log.Debugln("sent UL Packet from UE to UPF")
	return nil
}

func HandleDlMessage(gnbue *gnbctx.GnbUpUe, intfcMsg common.InterfaceMessage) (err error) {
	gnbue.Log.Debugln("handling DL Packet from UPF Worker")

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
			gnbue.Log.Infoln("received QFI value in downlink G-PDU:", extHdr.Qfi)
		}
	}

	ueDataMsg.Event = common.DL_UE_DATA_TRANSFER_EVENT
	// A UE that has stopped reading must not keep this worker from reaching its QUIT command.
	select {
	case gnbue.WriteUeChan <- ueDataMsg:
		gnbue.Log.Infoln("sent DL user data packet to UE")
	case <-gnbue.QuitRequested():
		gnbue.Log.Debugln("dropped a DL user data packet, the context is quitting")
	}

	return nil
}

// lastPktWaitTimeout bounds how long a quitting worker spends on the last data packets: handing its
// own to the UE and waiting for the UE's.
const lastPktWaitTimeout = 10 * time.Second

func HandleQuitEvent(gnbue *gnbctx.GnbUpUe, intfcMsg common.InterfaceMessage) (err error) {
	// The UE may have moved to another gNB (N2 handover) and not been able to hand over its last
	// packet, and it may not be reading either. Nothing is sent to this worker after that, so
	// everything here is bounded and the worker terminates either way.
	ctx, cancel := context.WithTimeout(context.Background(), lastPktWaitTimeout)
	defer cancel()

	// Uplink packets are read while the marker is handed over: the UE may be blocked sending them
	// and so unable to read the marker.
	if gnbue.WriteUeChan != nil {
		userDataMsg := &common.UserDataMessage{}
		userDataMsg.Event = common.LAST_DATA_PKT_EVENT
	send:
		for {
			select {
			case gnbue.WriteUeChan <- userDataMsg:
				break send
			case pkt := <-gnbue.ReadUlChan:
				if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
					gnbue.LastDataPktRecvd = true
				}
			case <-ctx.Done():
				gnbue.Log.Warnln("could not hand the last downlink data packet to the UE")
				break send
			}
		}
		gnbue.WriteUeChan = nil
	}

	// Drain all the messages until END MARKER is received.
	// This ensures that the transmitting go routine is not blocked while
	// sending data on this channel
	if !gnbue.LastDataPktRecvd {
	drain:
		for {
			select {
			case pkt := <-gnbue.ReadUlChan:
				if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
					gnbue.Log.Debugln("received last uplink data packet")
					break drain
				}
			case <-ctx.Done():
				gnbue.Log.Warnln("no last uplink data packet was received, terminating without it")
				break drain
			}
		}
	}

	gnbue.Gnb.DlTeidGenerator.FreeID(int64(gnbue.DlTeid))
	gnbue.Log.Infoln("Gnb User-plane UE Context terminated")

	return nil
}
