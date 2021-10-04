// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package pdusessworker

import (
	"gnbsim/common"
	"gnbsim/realue/context"
)

func SendUlMessage(gnbue *context.PduSession, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Traceln("Sending UL user data packet to gNb")

	return nil
}

func HandleDlMessage(gnbue *context.PduSession, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Traceln("Handling DL user data packet from gNb")

	return nil
}
