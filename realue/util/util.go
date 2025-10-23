// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"

	"github.com/omec-project/openapi/models"
)

const (
	SUPI_FORMAT          uint8 = 0x00 // imsi
	ID_TYPE              uint8 = 0x01 // suci
	PROTECTION_SCHEME_ID uint8 = 0x00 // null scheme
	PUBLIC_KEY_ID        uint8 = 0x00
	SUCI_LEN             uint8 = 22
)

var ROUTING_INDICATOR []uint8 = []uint8{0xf0, 0xff}

// encodeBCD encodes a string of decimal digits into telephony BCD (Binary Coded
// Decimal) format.
//
// Telephony BCD encoding specifics:
//   - Each byte encodes two digits: the second digit is placed in the upper
//     nibble (bits 4-7), and the first digit is placed in the lower nibble
//     (bits 0-3).
//   - If the input string has an odd number of digits, the final byte's upper
//     nibble is padded with 0xF.
//   - For example, "123" encodes to [0x21, 0xF3].
//   - Only decimal digits ('0'-'9') are allowed; any other character will
//     result in an error.
func encodeBCD(digits string) ([]byte, error) {
	for _, char := range digits {
		if char < '0' || char > '9' {
			return nil, fmt.Errorf("invalid character '%c' in digits string", char)
		}
	}

	digitBytes := []byte(digits)
	length := len(digitBytes)

	encodedLen := (length + 1) / 2
	encoded := make([]byte, encodedLen)

	for i := 0; i < length; i += 2 {
		firstDigit := digitBytes[i] - '0'

		var secondDigit byte
		if i+1 < length {
			secondDigit = digitBytes[i+1] - '0'
		} else {
			secondDigit = 0xF
		}

		// In telephony BCD, the second digit goes in the upper nibble
		// and the first digit goes in the lower nibble
		encoded[i/2] = (secondDigit << 4) | firstDigit
	}

	return encoded, nil
}

func SupiToSuci(supi string, plmnid *models.PlmnId) ([]byte, error) {
	supiExpectedPrefix := "imsi-" + plmnid.Mcc + plmnid.Mnc
	if !strings.HasPrefix(supi, supiExpectedPrefix) {
		return nil, fmt.Errorf(`invalid supi format, should start with "imsi-" + MCC + MNC`)
	}
	// extracting msin from supi
	msin := supi[len(supiExpectedPrefix):]

	suci := make([]uint8, 0, SUCI_LEN)
	// creating octet 4 of 5GS mobile identity info
	octet := (SUPI_FORMAT << 4) | ID_TYPE
	suci = append(suci, octet)

	bcdMcc, err := encodeBCD(plmnid.Mcc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode mcc in bcd format: %v", err)
	}
	suci = append(suci, bcdMcc...)

	bcdMnc, err := encodeBCD(plmnid.Mnc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode mnc in bcd format: %v", err)
	}
	suci = append(suci, bcdMnc...)
	suci = append(suci, ROUTING_INDICATOR...)
	suci = append(suci, PROTECTION_SCHEME_ID)
	suci = append(suci, PUBLIC_KEY_ID)

	bcdMsin, err := encodeBCD(msin)
	if err != nil {
		return nil, fmt.Errorf("failed to encode msin in bcd format: %v", err)
	}
	suci = append(suci, bcdMsin...)

	return suci, nil
}
