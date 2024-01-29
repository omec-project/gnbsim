// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"

	"github.com/omec-project/openapi/models"
	"github.com/yerden/go-util/bcd"
)

const (
	SUPI_FORMAT          uint8 = 0x00 // imsi
	ID_TYPE              uint8 = 0x01 // suci
	PROTECTION_SCHEME_ID uint8 = 0x00 // null scheme
	PUBLIC_KEY_ID        uint8 = 0x00
	SUCI_LEN             uint8 = 22
)

var ROUTING_INDICATOR []uint8 = []uint8{0xf0, 0xff}

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

	enc := bcd.NewEncoder(bcd.Telephony)
	bcdMcc := make([]byte, bcd.EncodedLen(len(plmnid.Mcc)))
	_, err := enc.Encode(bcdMcc, []byte(plmnid.Mcc))
	if err != nil {
		return nil, fmt.Errorf("failed to encode mcc in bcd format:%v", err)
	}
	suci = append(suci, bcdMcc...)

	bcdMnc := make([]byte, bcd.EncodedLen(len(plmnid.Mnc)))
	_, err = enc.Encode(bcdMnc, []byte(plmnid.Mnc))
	if err != nil {
		return nil, fmt.Errorf("failed to encode mnc in bcd format:%v", err)
	}
	suci = append(suci, bcdMnc...)
	suci = append(suci, ROUTING_INDICATOR...)
	suci = append(suci, PROTECTION_SCHEME_ID)
	suci = append(suci, PUBLIC_KEY_ID)

	bcdMsin := make([]byte, bcd.EncodedLen(len(msin)))
	_, err = enc.Encode(bcdMsin, []byte(msin))
	if err != nil {
		return nil, fmt.Errorf("failed to encode msin in bcd format:%v", err)
	}
	suci = append(suci, bcdMsin...)

	return suci, nil
}
