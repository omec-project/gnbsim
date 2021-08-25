// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package simue

import (
	"gnbsim/realue"
	"gnbsim/simue/context"
)

func Init(simue *context.SimUe) {
	go realue.Init(simue.RealUe)
	// Start Sim UE event generation/processing logic
}
