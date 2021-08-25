// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	gnbctx "gnbsim/gnodeb/context"
	intfc "gnbsim/interfacecommon"
	realuectx "gnbsim/realue/context"

	"github.com/free5gc/nas/security"
)

// SimUe controls the flow of messages between RealUe and GnbUe as per the test
// profile. It is the central entry point for all events
type SimUe struct {
	GnB    *gnbctx.GNodeB
	RealUe *realuectx.RealUe

	// SimUe writes messages to RealUE on this channel
	WriteRealUeChan chan *intfc.UuMessage

	// SimUe writes messages to GnbUE on this channel
	WriteGnbUeChan chan *intfc.UuMessage

	// SimUe reads messages from other entities on this channel
	// Entities can be RealUe, GnbUe etc.
	ReadChan chan *intfc.UuMessage
}

func NewSimUe(gnb *gnbctx.GNodeB) *SimUe {
	simue := SimUe{}
	simue.GnB = gnb
	simue.ReadChan = make(chan *intfc.UuMessage)
	simue.RealUe = realuectx.NewRealUeContext("imsi-2089300007487",
		security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		simue.ReadChan)
	simue.WriteRealUeChan = simue.RealUe.ReadChan

	return &simue
}
