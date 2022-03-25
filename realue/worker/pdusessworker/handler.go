// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package pdusessworker

import (
	"encoding/hex"
	"fmt"
	"net"

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

func SendUdpPacket(pduSess *realuectx.PduSession) (err error) {
	pduSess.Log.Traceln("Sending UL UDP message")
	ipSrcStr := pduSess.PduAddress.String()
	ipDstStr := pduSess.DefaultAs
	udpSrcPort := uint16(pduSess.DefaultAsSrcPort)
	udpDstPort := uint16(pduSess.DefaultAsDstPort)
	line := "Sending udp message from UE to upf"
	payload := []byte(line)
	/*
		ipSrc := net.ParseIP(ipSrcStr)
		if ipSrc == nil {
			pduSess.Log.Errorln("Invalid source IP: %v\n", ipSrc)
			return nil
		}

		ipDst := net.ParseIP(ipDstStr)
		if ipDst == nil {
			pduSess.Log.Errorln("Invalid destination IP: %v\n", ipDst)
			return nil
		}

		ipPkt := test.iphdr{
			vhl:   0x45,
			tos:   0,
			id:    0x1234,
			off:   0,
			ttl:   64,
			proto: unix.IPPROTO_UDP,
		}

		copy(ipPkt.src[:], ipSrc.To4())
		copy(ipPkt.dst[:], ipDst.To4())
		// iplen and csum set later

		udpPkt := test.udphdr{
			src: uint16(udpSrcPort),
			dst: uint16(udpDstPprt),
		}
		// ulen and csum set later

		udplen := 8 + len(payload)
		totalLen := 20 + udplen
		if totalLen > 0xffff {
			pduSess.Log.Errorln("message is too large to fit into a packet: %v > %v\n", totalLen, 0xffff)
			return nil
		}

		ipPkt.iplen = uint16(totalLen)
		ipPkt.checksum()

		// the kernel doesn't touch the UDP checksum, so we can either set it
		// correctly or leave it zero to indicate that we didn't use a checksum
		udpPkt.ulen = uint16(udplen)
		udpPkt.checksum(&ipPkt, payload)
		if showChkSum {
			pduSess.Log.Traceln("ip checksum: 0x%x, udp checksum: 0x%x\n", ipPkt.csum, udpPkt.csum)
		}

		var b bytes.Buffer
		err = binary.Write(&b, binary.BigEndian, &ipPkt)
		if err != nil {
			pduSess.Log.Errorln("Error encoding the IP header: %v\n", err)
			return err
		}
		err = binary.Write(&b, binary.BigEndian, &udpPkt)
		if err != nil {
			pduSess.Log.Errorln("Error encoding the UDP header: %v\n", err)
			return err
		}
		err = binary.Write(&b, binary.BigEndian, &payload)
		if err != nil {
			pduSess.Log.Errorln("Error encoding the payload: %v\n", err)
			return err
		}
	*/
	byteBuffer, err := test.BuildRawUdpIp(ipSrcStr, ipDstStr,
		udpSrcPort, udpDstPort, payload)
	if err != nil {
		pduSess.Log.Errorln("Error in building udp packet: %v\n", err)
		return err
	}

	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.UL_UE_DATA_TRANSFER_EVENT
	userDataMsg.Payload = byteBuffer
	pduSess.WriteGnbChan <- userDataMsg
	pduSess.TxDataPktCount++

	msg := &common.UuMessage{}
	msg.Event = common.DATA_PKT_GEN_SUCCESS_EVENT
	pduSess.WriteUeChan <- msg
	pduSess.Log.Traceln("Sent Data Packet Generation Success Event")

	pduSess.Log.Traceln("Sent UL UDP message")

	pduSess.Log.Traceln("%v bytes were sent\n", len(byteBuffer))

	return nil
}

func HandleInitEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	pduSess.WriteGnbChan = msg.CommChan
	pduSess.LastDataPktRecvd = false
	return nil
}

func SendIcmpEchoRequest(pduSess *realuectx.PduSession) (err error) {

	pduSess.Log.Traceln("Sending UL ICMP ping message")

	icmpPayload, err := hex.DecodeString("8c870d0000000000101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")
	if err != nil {
		pduSess.Log.Errorln("Failed to decode icmp hexString ")
		return
	}
	icmpPayloadLen := len(icmpPayload)
	pduSess.Log.Traceln("ICMP payload size:", icmpPayloadLen)

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
		return
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
		pduSess.Log.Errorln("Failed to marshal icmp message")
		return
	}

	payload := append(v4HdrBuf, b...)

	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.UL_UE_DATA_TRANSFER_EVENT
	userDataMsg.Payload = payload
	pduSess.WriteGnbChan <- userDataMsg
	pduSess.TxDataPktCount++

	pduSess.Log.Traceln("Sent UL ICMP ping message")

	return nil
}

func HandleUdpMessage(pduSess *realuectx.PduSession,
	udpPkt []byte) (err error) {
	fmt.Printf("udp message handling not yet supported")
	return nil
}

func HandleIcmpMessage(pduSess *realuectx.PduSession,
	icmpPkt []byte) (err error) {
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

		pduSess.Log.Infof("Received ICMP Echo Reply, ID:%v, Seq:%v",
			echpReply.ID, echpReply.Seq)

		pduSess.RxDataPktCount++
		if pduSess.TxDataPktCount < pduSess.ReqDataPktCount {
			SendIcmpEchoRequest(pduSess)
		} else {
			msg := &common.UuMessage{}
			msg.Event = common.DATA_PKT_GEN_SUCCESS_EVENT
			pduSess.WriteUeChan <- msg
			pduSess.Log.Traceln("Sent Data Packet Generation Success Event")
		}
	default:
		return fmt.Errorf("unsupported icmp message type:%v", icmpMsg.Type)
	}

	return nil
}

func HandleDlMessage(pduSess *realuectx.PduSession,
	msg common.InterfaceMessage) (err error) {

	pduSess.Log.Traceln("Handling DL user data packet from gNb")

	if msg.GetEventType() == common.LAST_DATA_PKT_EVENT {
		pduSess.Log.Debugln("Received last downlink data packet")
		pduSess.LastDataPktRecvd = true
		return nil
	}

	dataMsg := msg.(*common.UserDataMessage)

	if dataMsg.Qfi != nil {
		pduSess.Log.Infoln("Received QFI value in downlink user data packet:", *dataMsg.Qfi)
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
	case 17:
		err = HandleUdpMessage(pduSess, dataMsg.Payload[ipv4Hdr.Len:])
		if err != nil {
			return fmt.Errorf("failed to handle udp message:%v", err)
		}
	default:
		return fmt.Errorf("unsupported ipv4 protocol:%v", ipv4Hdr.Protocol)
	}

	return nil
}

func HandleIcmpPktGenRequestEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage) (err error) {
	cmd := intfcMsg.(*common.UeMessage)
	pduSess.ReqDataPktCount = cmd.UserDataPktCount
	pduSess.DefaultAs = cmd.DefaultAs
	err = SendIcmpEchoRequest(pduSess)
	if err != nil {
		return fmt.Errorf("failed to send icmp echo req:%v", err)
	}
	return nil
}

func HandleUdpPktGenRequestEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage) (err error) {
	cmd := intfcMsg.(*common.UeMessage)
	pduSess.ReqDataPktCount = cmd.UserDataPktCount
	pduSess.DefaultAs = cmd.DefaultAs
	pduSess.DefaultAsSrcPort = cmd.DefaultAsSrcPort
	pduSess.DefaultAsDstPort = cmd.DefaultAsDstPort
	err = SendUdpPacket(pduSess)
	if err != nil {
		return fmt.Errorf("failed to send icmp echo req:%v", err)
	}
	return nil
}

func HandleConnectionReleaseRequestEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage) (err error) {

	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.LAST_DATA_PKT_EVENT
	pduSess.WriteGnbChan <- userDataMsg
	// Releasing the reference so as to be freed by Garbage Collector
	pduSess.WriteGnbChan = nil
	return nil
}

func HandleQuitEvent(pduSess *realuectx.PduSession,
	intfcMsg common.InterfaceMessage) (err error) {

	if pduSess.WriteGnbChan != nil {
		userDataMsg := &common.UserDataMessage{}
		userDataMsg.Event = common.LAST_DATA_PKT_EVENT
		pduSess.WriteGnbChan <- userDataMsg
		pduSess.WriteGnbChan = nil
	}

	// Drain all the messages until END MARKER is received.
	// This ensures that the transmitting go routine is not blocked while
	// sending data on this channel
	if pduSess.LastDataPktRecvd != true {
		for pkt := range pduSess.ReadDlChan {
			if pkt.GetEventType() == common.LAST_DATA_PKT_EVENT {
				pduSess.Log.Debugln("Received last downlink data packet")
				break
			}
		}
	}

	pduSess.WriteUeChan = nil
	pduSess.Log.Infoln("Pdu Session terminated")

	return nil
}
