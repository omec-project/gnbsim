// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package util

import (
	"gnbsim/common"
	simueCtx "gnbsim/simue/context"
)

func SendToSimUe(simUe *simueCtx.SimUe, event common.EventType) {
	msg := &common.ProfileMessage{}
	msg.Event = event
	simUe.ReadChan <- msg
}
