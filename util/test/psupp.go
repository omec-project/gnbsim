// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package test

import (
	"fmt"
	"gnbsim/logger"
)

// This file holds utility functions for PDU Session User Plane Protocol
// TS 38.415

const (
	UL_PDU_SESS_INFO_PDU uint8 = 1
	QFI_BIT_MASK         uint8 = 0x3f
)

func BuildUlPduSessInformation(qfi uint8) []uint8 {

	pdu := make([]uint8, 2, 2)
	pdu[0] = UL_PDU_SESS_INFO_PDU << 4
	pdu[1] = qfi
	return pdu
}

func DecodeDlPduSessInformation(pdu []uint8) (qfi uint8, err error) {
	if pdu != nil {
		return qfi, fmt.Errorf("pdu is null")
	}

	// Bit 7,6,5 and 4 for PDU type (0 for Dl PDU Session Info type pdu)
	if (pdu[0] & 0xf0) != 0 {
		return qfi, fmt.Errorf("invalid pdu type")
	}

	for i, octet := range pdu {
		switch i {
		case 1:
			qfi = octet & QFI_BIT_MASK
		default:
			logger.PsuppLog.Warnln("Field not supported")
			return
		}
	}

	return
}
