// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/omec-project/gnbsim/logger"
)

const (
	/* GTPv1 Header Flags Spec 3GPP TS-29281 */
	FLAG_GTP_VERSION_1     uint8 = 0x20
	FLAG_PROTOCOL_TYPE_GTP uint8 = 0x10
	FLAG_EXT_HEADER        uint8 = 0x04
	FLAG_SEQ_NUM           uint8 = 0x02
	FLAG_NPDU_NUM          uint8 = 0x01
	FLAG_REQUIRED          uint8 = (FLAG_GTP_VERSION_1 | FLAG_PROTOCOL_TYPE_GTP)
	FLAG_OPTIONAL          uint8 = (FLAG_EXT_HEADER | FLAG_SEQ_NUM | FLAG_NPDU_NUM)

	/* GTPv1 Message Types Spec 3GPP TS-29281 */
	TYPE_GPDU uint8 = 0xff

	/* GTPv1 IE Types Spec 3GPP TS-29281 */
	TEID_DATA_IE      uint8 = 0x10
	GTPU_PEER_ADDR_IE uint8 = 0x85

	/* Involves length of the mandatory GTP-U Header */
	GTPU_HEADER_LENGTH uint16 = 8

	/* involves length of Sequence number NPDU Number and Next Extension Header
	fields */
	OPT_GTPU_HEADER_LENGTH uint16 = 4

	PDU_SESS_CONTAINER_EXT_HEADER_TYPE uint8 = 0x85
)

type GtpHdr struct {
	Flags uint8 // Version(3-bits), Protocol Type(1-bit), Extension Header flag(1-bit),
	// Sequence Number flag(1-bit), N-PDU number flag(1-bit)
	MsgType uint8  // Message Type
	Len     uint16 // Total Length
	Teid    uint32 // Tunnel Endpoint Identifier
}

type GtpHdrOpt struct {
	SeqNum      uint16 // Sequence Number
	NpduNum     uint8  // N-PDU Number
	NextHdrType uint8  // Next Extenstion Header Type
}

type GtpPdu struct {
	Hdr     *GtpHdr
	OptHdr  *GtpHdrOpt
	Payload []uint8
}

type PduSessContainerExtHeader struct {
	Qfi               uint8
	NextExtHeaderType uint8
}

func BuildGTPv1Header(extHdrFlag bool, snFlag bool, nPduFlag bool,
	nExtHdrType uint8, sn uint16, nPduNum uint8, msgType uint8,
	payloadLen uint16, teID uint32,
) ([]byte, error) {
	var optHdrPresent bool

	/* Setting GTP-U header flags */
	var flags uint8 = FLAG_GTP_VERSION_1 | FLAG_PROTOCOL_TYPE_GTP
	if extHdrFlag {
		flags |= FLAG_EXT_HEADER
		optHdrPresent = true
	}
	if snFlag {
		flags |= FLAG_SEQ_NUM
		optHdrPresent = true
	}
	if nPduFlag {
		flags |= FLAG_NPDU_NUM
		optHdrPresent = true
	}

	if optHdrPresent {
		payloadLen += OPT_GTPU_HEADER_LENGTH
	}

	/* Populating GTP-U header */
	ghdr := GtpHdr{
		Flags:   flags,
		MsgType: msgType,
		Len:     payloadLen,
		Teid:    teID,
	}

	var b bytes.Buffer
	err := binary.Write(&b, binary.BigEndian, &ghdr)
	if err != nil {
		return nil, err
	}

	/* Populating optional fields if present */
	if optHdrPresent {
		ghdropt := GtpHdrOpt{
			SeqNum:      sn,
			NpduNum:     nPduNum,
			NextHdrType: nExtHdrType,
		}
		err = binary.Write(&b, binary.BigEndian, &ghdropt)
		if err != nil {
			return nil, err
		}
	}
	bb := b.Bytes()
	return bb, nil
}

func DecodeGTPv1Header(pkt []byte) (gtpPdu *GtpPdu, err error) {
	gtpPdu = &GtpPdu{}
	gtpPdu.Hdr = &GtpHdr{}

	buf := bytes.NewReader(pkt)
	err = binary.Read(buf, binary.BigEndian, gtpPdu.Hdr)
	if err != nil {
		return nil, err
	}

	if (gtpPdu.Hdr.Flags & FLAG_REQUIRED) != FLAG_REQUIRED {
		err = fmt.Errorf("invalid gtp version or protocol type")
		return nil, err
	}

	logger.GtpLog.Traceln("Header field - Length:", gtpPdu.Hdr.Len)
	logger.GtpLog.Traceln("Header field - TEID:", gtpPdu.Hdr.Teid)

	payloadStart := GTPU_HEADER_LENGTH
	payloadEnd := gtpPdu.Hdr.Len + GTPU_HEADER_LENGTH

	if (gtpPdu.Hdr.Flags & FLAG_OPTIONAL) != 0 {
		logger.GtpLog.Traceln("Optional header present")
		gtpPdu.OptHdr = &GtpHdrOpt{}
		err = binary.Read(buf, binary.BigEndian, gtpPdu.OptHdr)
		if err != nil {
			return nil, err
		}
		payloadStart += OPT_GTPU_HEADER_LENGTH
	}

	if len(pkt) != int(payloadEnd) {
		err = fmt.Errorf("invalid payload length")
		return nil, err
	}

	gtpPdu.Payload = pkt[payloadStart:payloadEnd]
	return gtpPdu, nil
}

func BuildPduSessContainerExtHeader(qfi uint8) []uint8 {
	pdu := BuildUlPduSessInformation(qfi)

	// Octet count for Ext Header Length + Ext Header Count + Next Ext Header Type
	octetCount := 2 + len(pdu)

	// The length of Extension Header shall be defined in variable length of 4
	// octets (5.2.1 TS 29.281)
	if r := (octetCount % 4); r != 0 {
		spareOctetCount := 4 - r
		octetCount += spareOctetCount
		spareOctets := make([]uint8, spareOctetCount)
		pdu = append(pdu, spareOctets...)
	}

	extHeaderLen := uint8(octetCount / 4)

	pduSessContainer := make([]uint8, 0, octetCount)
	pduSessContainer = append(pduSessContainer, extHeaderLen)
	pduSessContainer = append(pduSessContainer, pdu...)
	// appending NextExtHeaderType as 0
	pduSessContainer = append(pduSessContainer, 0)

	return pduSessContainer
}

// Currently processing QFI only
func DecodePduSessContainerExtHeader(pkt []uint8) (payload []uint8,
	extHdr *PduSessContainerExtHeader, err error,
) {
	bufLen := len(pkt)
	if bufLen == 0 {
		err = fmt.Errorf("extension header is nil")
		return nil, nil, err
	}

	logger.GtpLog.Info("PDU Session Container Extension header length:", pkt[0])

	// First octet is Extension Header Length
	octetCount := pkt[0] * 4
	if octetCount == 0 || bufLen < int(octetCount) {
		err = fmt.Errorf("incomplete extension header - buffer length: %v, extension header length value: %v", bufLen, pkt[0])
		return nil, nil, err
	}

	extHdr = &PduSessContainerExtHeader{}
	// Last octet of Extension header is Next Extension Header Type
	extHdr.Qfi, err = DecodeDlPduSessInformation(pkt[1:(octetCount - 1)])
	if err != nil {
		err = fmt.Errorf("failed to decode downlink pdu sessioon information:%v", err)
		return nil, nil, err
	}

	extHdr.NextExtHeaderType = pkt[octetCount-1]

	if bufLen > int(octetCount) {
		payload = pkt[octetCount:]
	}

	return payload, extHdr, nil
}

func BuildGpduMessage(payload []byte, teID uint32) ([]byte, error) {
	pduSessContainer := BuildPduSessContainerExtHeader(9)

	/* UE needs to ensure its payload length value should not exceed 2 bytes */
	payloadLen := uint16(len(payload) + len(pduSessContainer))

	b, err := BuildGTPv1Header(true, false, false,
		PDU_SESS_CONTAINER_EXT_HEADER_TYPE, 0, 0, TYPE_GPDU, payloadLen, teID)
	if err != nil {
		return nil, err
	}

	b = append(b, pduSessContainer...)
	b = append(b, payload...)
	return b, nil
}
