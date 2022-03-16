// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package idrange

import (
	"math/rand"
	"time"

	"github.com/omec-project/gnbsim/logger"
)

/*
	The Range selector uses MSB of the ID to mark the current range
	eg.Given the ID is of 4 byte and range selector is of 1 byte then,
	for a range selector value of 25(dec) the start and end IDs will be
	0x19000000 and 0x19ffffff respectively
*/

const (
	/* Maximum bit length of ID */
	MAX_ID_BITS uint8 = 32

	/* Maximum bit length of range selector */
	MAX_RANGE_BITS uint8 = 8
)

// TODO : add UT
func GetIdRange() (start, end uint32) {
	maxRangeSelectVal := (1 << MAX_RANGE_BITS) - 1
	idBitCount := MAX_ID_BITS - MAX_RANGE_BITS

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rangeSelectVal := uint32(r.Intn(maxRangeSelectVal))
	logger.GNodeBLog.Infoln("Current range selector value:", rangeSelectVal)

	// Shifting Range Selector value to MSB
	start = rangeSelectVal << idBitCount

	// Next range start vlalue subtracted by 1
	end = ((rangeSelectVal + 1) << idBitCount) - 1

	if start == 0 {
		start = 1
	}

	logger.GNodeBLog.Infoln("Current ID range start:", start, "end:", end)
	return
}
