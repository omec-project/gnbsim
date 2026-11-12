// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package pdusessworker

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/omec-project/gnbsim/common"
	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/gnbsim/util/test"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	ICMP_HEADER_LEN int = 8

	/*ipv4 package requires ipv4 header length in terms of number of bytes,
	  however it later converts it into number of 32 bit words
	*/
	IPV4_MIN_HEADER_LEN int = 20
)

// lastPktWaitTimeout bounds how long a quitting session waits for the gNB's last data packet.
var lastPktWaitTimeout = 10 * time.Second

// ulSendTimeout bounds how long a one-off uplink packet waits for room in the gNB's queue.
var ulSendTimeout = 5 * time.Second

// lastPktSendTimeout bounds the hand-over of the last packet marker to a gNB that is not reading.
const lastPktSendTimeout = 2 * time.Second

func HandleInitEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	// Moving to another gNB (N2 handover): the user plane worker of the one being left waits for
	// the last uplink packet marker before it can terminate, which the quit would have sent.
	if old := pduSess.WriteGnbChan; old != nil && old != msg.CommChan {
		// a generator would go on sending to the gNB that is being left, after its marker
		stopGenerator(pduSess)
		lastPkt := &common.UserDataMessage{}
		lastPkt.Event = common.LAST_DATA_PKT_EVENT
		select {
		case old <- lastPkt:
		case <-time.After(lastPktSendTimeout):
			pduSess.Log.Warnln("could not hand the last uplink packet marker to the previous gNB")
		}
	}
	pduSess.WriteGnbChan = msg.CommChan
	pduSess.LastDataPktRecvd = false
	return nil
}

func SendIcmpEchoRequest(pduSess *realuectx.PduSession) (err error) {
	return sendIcmpEchoRequest(pduSess, nil)
}

// sendIcmpEchoRequest sends one echo request. A send that is blocked gives way to stop, which is
// the generator's way of being cancelled; a nil stop is never ready.
func sendIcmpEchoRequest(pduSess *realuectx.PduSession, stop <-chan struct{}) (err error) {
	// After an AN release the session has no gNB to send to. The select below would then wait on a
	// nil channel, and with it this worker, which could no longer process its quit.
	if pduSess.WriteGnbChan == nil {
		return fmt.Errorf("pdu session %d is not connected to a gNB", pduSess.PduSessId)
	}
	pduSess.Log.Debugln("sending UL ICMP ping message")

	icmpPayload, err := hex.DecodeString("8c870d0000000000101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")
	if err != nil {
		pduSess.Log.Errorln("failed to decode icmp hexString ")
		return err
	}
	icmpPayloadLen := len(icmpPayload)
	pduSess.Log.Debugln("ICMP payload size:", icmpPayloadLen)

	ipv4hdr := ipv4.Header{
		Version:  4,
		Len:      IPV4_MIN_HEADER_LEN,
		Protocol: 1,
		Flags:    0,
		TotalLen: IPV4_MIN_HEADER_LEN + ICMP_HEADER_LEN + icmpPayloadLen,
		TTL:      64,
		Src:      pduSess.PduAddress,                   // ue IP address
		Dst:      net.ParseIP(pduSess.DefaultAs).To4(), // upstream router interface connected to Gi
		ID:       1,
	}
	checksum := test.CalculateIpv4HeaderChecksum(&ipv4hdr)
	ipv4hdr.Checksum = int(checksum)

	v4HdrBuf, err := ipv4hdr.Marshal()
	if err != nil {
		pduSess.Log.Errorln("ipv4hdr header marshal failed")
		return err
	}

	icmpMsg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 12394, Seq: pduSess.GetNextSeqNum(),
			Data: icmpPayload,
		},
	}
	b, err := icmpMsg.Marshal(nil)
	if err != nil {
		pduSess.Log.Errorln("failed to marshal icmp message")
		return err
	}

	payload := append(v4HdrBuf, b...)

	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.UL_UE_DATA_TRANSFER_EVENT
	userDataMsg.Payload = payload
	// A generator is cancelled through stop. A one-off request has no such signal, and it runs on
	// the session's own event loop: blocked on a full uplink queue it would keep the session from
	// ever processing its quit, so the wait is bounded.
	var timeout <-chan time.Time
	if stop == nil {
		timer := time.NewTimer(ulSendTimeout)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case pduSess.WriteGnbChan <- userDataMsg:
	case <-stop:
		return fmt.Errorf("packet generation was stopped")
	case <-timeout:
		return fmt.Errorf("the gNB is not reading the uplink of pdu session %d", pduSess.PduSessId)
	}
	pduSess.TxDataPktCount++

	pduSess.Log.Debugln("sent UL ICMP ping message")

	return nil
}

func HandleIcmpMessage(pduSess *realuectx.PduSession,
	icmpPkt []byte,
) (err error) {
	icmpMsg, err := icmp.ParseMessage(1, icmpPkt)
	if err != nil {
		return fmt.Errorf("failed to parse icmp message:%v", err)
	}

	switch icmpMsg.Type {
	case ipv4.ICMPTypeEchoReply:
		echpReply := icmpMsg.Body.(*icmp.Echo)
		if echpReply == nil {
			return fmt.Errorf("icmp echo reply is nil")
		}

		pduSess.Log.Infof("received ICMP Echo Reply, ID:%v, Seq:%v",
			echpReply.ID, echpReply.Seq)

		pduSess.RxDataPktCount++
		if pduSess.ReqDataPktInt == 0 {
			if pduSess.TxDataPktCount < pduSess.ReqDataPktCount {
				err := SendIcmpEchoRequest(pduSess)
				if err != nil {
					return fmt.Errorf("failed to send icmp message:%v", err)
				}
			} else {
				msg := &common.UuMessage{}
				msg.Event = common.DATA_PKT_GEN_SUCCESS_EVENT
				pduSess.WriteUeChan <- msg
				pduSess.Log.Debugln("sent Data Packet Generation Success Event")
			}
		}
	default:
		return fmt.Errorf("unsupported icmp message type:%v", icmpMsg.Type)
	}

	return nil
}

func HandleDlMessage(pduSess *realuectx.PduSession,
	msg common.InterfaceMessage,
) (err error) {
	pduSess.Log.Debugln("handling DL user data packet from gNb")

	if msg.GetEventType() == common.LAST_DATA_PKT_EVENT {
		pduSess.Log.Debugln("received last downlink data packet")
		pduSess.LastDataPktRecvd = true
		return nil
	}

	dataMsg := msg.(*common.UserDataMessage)

	if dataMsg.Qfi != nil {
		pduSess.Log.Infoln("received QFI value in downlink user data packet:", *dataMsg.Qfi)
	}

	ipv4Hdr, err := ipv4.ParseHeader(dataMsg.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse ipv4 header:%v", err)
	}

	switch ipv4Hdr.Protocol {
	/* Currently supporting ICMP protocol */
	case 1:
		err = HandleIcmpMessage(pduSess, dataMsg.Payload[ipv4Hdr.Len:])
		if err != nil {
			return fmt.Errorf("failed to handle icmp message:%v", err)
		}
	default:
		return fmt.Errorf("unsupported ipv4 protocol:%v", ipv4Hdr.Protocol)
	}

	return nil
}

func HandleDataPktGenRequestEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage,
) (err error) {
	cmd := intfcMsg.(*common.UeMessage)
	// After an AN release there is no gNB to send to. Refused here, where the error reaches the
	// profile as the result of the request, and not in a generator that could only log it.
	if pduSess.WriteGnbChan == nil {
		return fmt.Errorf("pdu session %d is not connected to a gNB", pduSess.PduSessId)
	}
	// A generator that is still running reads these settings, so it goes before they change. This
	// also keeps a one-off request from running next to a timed generator.
	stopGenerator(pduSess)
	pduSess.ReqDataPktCount = cmd.UserDataPktCount
	pduSess.ReqDataPktInt = cmd.UserDataPktInterval
	pduSess.DefaultAs = cmd.DefaultAs
	if pduSess.ReqDataPktInt == 0 {
		err = SendIcmpEchoRequest(pduSess)
		if err != nil {
			return fmt.Errorf("failed to send icmp echo req:%v", err)
		}
	} else {
		stop := make(chan struct{})
		pduSess.GenStop = stop
		pduSess.GenWg.Go(func() {
			for pduSess.TxDataPktCount < pduSess.ReqDataPktCount {
				err := sendIcmpEchoRequest(pduSess, stop)
				if err != nil {
					pduSess.Log.Errorf("failed to send icmp echo req: %v", err)
					return // Exit the goroutine on error
				}
				select {
				case <-time.After(time.Duration(pduSess.ReqDataPktInt) * time.Second):
				case <-stop:
					return
				}
			}
			msg := &common.UuMessage{}
			msg.Event = common.DATA_PKT_GEN_SUCCESS_EVENT
			select {
			case pduSess.WriteUeChan <- msg:
				pduSess.Log.Debugln("sent Data Packet Generation Success Event")
			case <-stop:
			}
		})
	}
	return nil
}

// stopGenerator stops the packet generator of the session, if there is one, and waits until it
// has returned, so that it is gone before the channels it sends on are cleared.
func stopGenerator(pduSess *realuectx.PduSession) {
	if pduSess.GenStop != nil {
		close(pduSess.GenStop)
		pduSess.GenStop = nil
	}
	pduSess.GenWg.Wait()
}

func HandleConnectionReleaseRequestEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage,
) (err error) {
	// the generator uses the channel that is cleared below
	stopGenerator(pduSess)

	// Already released, or never connected: there is no one to tell, and a send on a nil channel
	// would block this worker for good.
	if pduSess.WriteGnbChan == nil {
		return nil
	}
	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.LAST_DATA_PKT_EVENT
	// bounded, and the downlink is read meanwhile: the gNB may be blocked sending it to this
	// worker, and so unable to read the marker
	ctx, cancel := context.WithTimeout(context.Background(), lastPktSendTimeout)
	defer cancel()
send:
	for {
		select {
		case pduSess.WriteGnbChan <- userDataMsg:
			break send
		case pkt := <-pduSess.ReadDlChan:
			if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
				pduSess.LastDataPktRecvd = true
			}
		case <-ctx.Done():
			pduSess.Log.Warnln("could not hand the last uplink data packet to the gNB")
			break send
		}
	}
	// Releasing the reference so as to be freed by Garbage Collector
	pduSess.WriteGnbChan = nil
	return nil
}

func HandleQuitEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage,
) (err error) {
	// the generator uses the channels that are cleared below
	stopGenerator(pduSess)

	// The gNB's user plane worker may never have been started, for instance because the data
	// bearer setup response did not reach the gNB before the UE quit, and it may not be reading.
	// Everything here is bounded, and the session terminates either way.
	ctx, cancel := context.WithTimeout(context.Background(), lastPktWaitTimeout)
	defer cancel()

	// Downlink packets are read while the marker is handed over: the gNB may be blocked sending
	// them and so unable to read the marker.
	if pduSess.WriteGnbChan != nil {
		userDataMsg := &common.UserDataMessage{}
		userDataMsg.Event = common.LAST_DATA_PKT_EVENT
	send:
		for {
			select {
			case pduSess.WriteGnbChan <- userDataMsg:
				break send
			case pkt := <-pduSess.ReadDlChan:
				if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
					pduSess.LastDataPktRecvd = true
				}
			case <-ctx.Done():
				pduSess.Log.Warnln("could not hand the last uplink data packet to the gNB")
				break send
			}
		}
		pduSess.WriteGnbChan = nil
	}

	// Drain all the messages until END MARKER is received.
	// This ensures that the transmitting go routine is not blocked while
	// sending data on this channel
	//
	// The marker comes from the gNB's user plane worker, which may not exist (see above).
	if !pduSess.LastDataPktRecvd {
	drain:
		for {
			select {
			case pkt := <-pduSess.ReadDlChan:
				if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
					pduSess.Log.Debugln("received last downlink data packet")
					break drain
				}
			case <-ctx.Done():
				pduSess.Log.Warnln("no last downlink data packet was received, terminating without it")
				break drain
			}
		}
	}

	pduSess.WriteUeChan = nil
	pduSess.Log.Infoln("Pdu Session terminated")

	return nil
}
