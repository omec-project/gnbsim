// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
)

// This file holds utility functions for PDU Session User Plane Protocol
// TS 38.415

const (
	DL_PDU_SESS_INFO_PDU_TYPE uint8 = 0
	UL_PDU_SESS_INFO_PDU_TYPE uint8 = 1
	QFI_BIT_MASK              uint8 = 0x3f
)

func BuildUlPduSessInformation(qfi uint8) []uint8 {
	pdu := make([]uint8, 2)
	pdu[0] = UL_PDU_SESS_INFO_PDU_TYPE << 4
	pdu[1] = qfi
	return pdu
}

func DecodeDlPduSessInformation(pdu []uint8) (qfi uint8, err error) {
	if len(pdu) < 2 {
		err = fmt.Errorf("incomplete pdu")
		return
	}

	// Bit 7,6,5 and 4 for PDU type (0 for Dl PDU Session Info type pdu)
	if (pdu[0] & 0xf0) != DL_PDU_SESS_INFO_PDU_TYPE {
		err = fmt.Errorf("invalid pdu type")
		return
	}

	qfi = pdu[1] & QFI_BIT_MASK

	return
}
