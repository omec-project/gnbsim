// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package pdusessworker

import (
	"encoding/hex"
	"fmt"
	"gnbsim/common"
	"gnbsim/realue/context"
	"gnbsim/util/test"
	"net"

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

func SendIcmpEchoRequest(pduSess *context.PduSession) (err error) {

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
		Src:      pduSess.PduAddress,                 // ue IP address
		Dst:      net.ParseIP("192.168.250.1").To4(), // upstream router interface connected to Gi
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

func HandleIcmpMessage(pduSess *context.PduSession,
	icmpPkt []byte) (err error) {
	icmpMsg, err := icmp.ParseMessage(1, icmpPkt)
	if err != nil {
		pduSess.Log.Errorln("icmp.ParseMessage() returned:", err)
		return fmt.Errorf("invalid icmp message")
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

func HandleDlMessage(pduSess *context.PduSession,
	msg common.InterfaceMessage) (err error) {

	pduSess.Log.Traceln("Handling DL user data packet from gNb")

	dataMsg := msg.(*common.UserDataMessage)

	ipv4Hdr, err := ipv4.ParseHeader(dataMsg.Payload)
	if err != nil {
		pduSess.Log.Errorln("ipv4.ParseHeader() returned:", err)
		return fmt.Errorf("invalid ipv4 header")
	}

	switch ipv4Hdr.Protocol {
	/* Currently supporting ICMP protocol */
	case 1:
		err = HandleIcmpMessage(pduSess, dataMsg.Payload[ipv4Hdr.Len:])
		if err != nil {
			pduSess.Log.Errorln("HandleIcmpMessage() returned:", err)
			return fmt.Errorf("failed to handle icmp message")
		}
	default:
		return fmt.Errorf("unsupported ipv4 protocol:%v", ipv4Hdr.Protocol)
	}

	return nil
}

func HandleDataPktGenRequestEvent(pduSess *context.PduSession,
	intfcMsg common.InterfaceMessage) (err error) {
	cmd := intfcMsg.(*common.UeMessage)
	pduSess.ReqDataPktCount = cmd.UserDataPktCount
	err = SendIcmpEchoRequest(pduSess)
	if err != nil {
		pduSess.Log.Errorln("SendIcmpEchoRequest() returned:", err)
		return fmt.Errorf("failed to send icmp echo req")
	}
	return nil
}
