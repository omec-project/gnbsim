// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	gnbctx "gnbsim/gnodeb/context"
	intfc "gnbsim/interfacecommon"
	"gnbsim/logger"
	realuectx "gnbsim/realue/context"

	"github.com/free5gc/nas/security"
	"github.com/sirupsen/logrus"
)

// SimUe controls the flow of messages between RealUe and GnbUe as per the test
// profile. It is the central entry point for all events
type SimUe struct {
	Supi   string
	GnB    *gnbctx.GNodeB
	RealUe *realuectx.RealUe

	// SimUe writes messages to RealUE on this channel
	WriteRealUeChan chan *intfc.UuMessage

	// SimUe writes messages to GnbUE on this channel
	WriteGnbUeChan chan intfc.InterfaceMessage

	// SimUe reads messages from other entities on this channel
	// Entities can be RealUe, GnbUe etc.
	ReadChan chan *intfc.UuMessage

	/* logger */
	Log *logrus.Entry
}

func NewSimUe(gnb *gnbctx.GNodeB) *SimUe {
	supi := "imsi-2089300007487"
	suci := []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00,
		0x47, 0x78}

	simue := SimUe{}
	simue.GnB = gnb
	simue.Supi = supi
	simue.ReadChan = make(chan *intfc.UuMessage)
	simue.RealUe = realuectx.NewRealUeContext(supi,
		security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		simue.ReadChan, suci)
	simue.WriteRealUeChan = simue.RealUe.ReadChan

	simue.Log = logger.RealUeLog.WithField(logger.FieldSupi, supi)

	simue.Log.Debugln("Created new context")
	return &simue
}
