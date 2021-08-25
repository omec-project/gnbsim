// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package realue

import (
	intfc "gnbsim/interfacecommon"
	"gnbsim/realue/context"
	"log"
)

func Init(ue *context.RealUe) {
	for msg := range ue.ReadChan {
		err := HandleMessage(ue, msg)
		if err != nil {
			log.Println(err)
		}
	}
}

func HandleMessage(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	// Handle NAS Message generation/processing logic here
	return nil
}
