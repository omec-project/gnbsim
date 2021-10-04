// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package test

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	/* GTPv1 Header Flags Spec 3GPP TS-29281 */
	FLAG_GTP_VERSION_1     uint8 = 0x20
	FLAG_PROTOCOL_TYPE_GTP uint8 = 0x10
	FLAG_EXT_HEADER        uint8 = 0x04
	FLAG_SEQ_NUM           uint8 = 0x02
	FLAG_NPDU_NUM          uint8 = 0x01

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
)

type GtpHdr struct {
	Flags uint8 //Version(3-bits), Protocal Type(1-bit), Extension Header flag(1-bit),
	// Sequence Number flag(1-bit), N-PDU number flag(1-bit)
	MsgType uint8  //Message Type
	Len     uint16 //Total Length
	Teid    uint32 //Tunnel Endpoint Identifier
}

type GtpHdrOpt struct {
	SeqNum      uint16 //Sequence Number
	NpduNum     uint8  //N-PDU Number
	NextHdrType uint8  //Next Extenstion Header Type
}

func BuildGTPv1Header(extHdrFlag bool, snFlag bool, nPduFlag bool,
	nExtHdrType uint8, sn uint16, nPduNum uint8, msgType uint8,
	payloadLen uint16, teID uint32) ([]byte, error) {

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
		payloadLen += uint16(OPT_GTPU_HEADER_LENGTH)
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

func DecodeGTPv1Header(pkt []byte, hdr *GtpHdr, optHdr *GtpHdrOpt) (payload []byte,
	err error) {
	buf := bytes.NewReader(pkt)
	err = binary.Read(buf, binary.LittleEndian, hdr)
	if err != nil {
		return nil, err
	}

	requiredFlags := FLAG_GTP_VERSION_1 | FLAG_PROTOCOL_TYPE_GTP
	optFlags := FLAG_EXT_HEADER | FLAG_SEQ_NUM | FLAG_NPDU_NUM

	if (hdr.Flags & requiredFlags) != requiredFlags {
		return nil, fmt.Errorf("invalid gtp version or protocol type")
	}

	payloadStart := GTPU_HEADER_LENGTH
	payloadEnd := hdr.Len + GTPU_HEADER_LENGTH

	if (hdr.Flags & optFlags) != 0 {
		err = binary.Read(buf, binary.LittleEndian, optHdr)
		if err != nil {
			return nil, err
		}
		payloadStart += OPT_GTPU_HEADER_LENGTH
		payloadEnd += OPT_GTPU_HEADER_LENGTH
	}

	if len(pkt) != int(payloadEnd) {
		return nil, fmt.Errorf("invalid payload length")
	}

	return pkt[payloadStart:payloadEnd], nil
}

func BuildGpduMessage(payload []byte, teID uint32) ([]byte, error) {
	/* UE needs to ensure its payload length should not exceed 2 bytes */
	payloadLen := uint16(len(payload))
	b, err := BuildGTPv1Header(false, false, false, 0, 0, 0, TYPE_GPDU,
		payloadLen, teID)
	if err != nil {
		return nil, err
	}
	b = append(b, payload...)
	return b, nil
}
