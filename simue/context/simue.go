// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"gnbsim/common"
	gnbctx "gnbsim/gnodeb/context"
	"gnbsim/logger"
	profctx "gnbsim/profile/context"
	realuectx "gnbsim/realue/context"

	"github.com/free5gc/nas/security"
	"github.com/sirupsen/logrus"
)

// SimUe controls the flow of messages between RealUe and GnbUe as per the test
// profile. It is the central entry point for all events
type SimUe struct {
	Supi       string
	GnB        *gnbctx.GNodeB
	RealUe     *realuectx.RealUe
	ProfileCtx *profctx.Profile
	Procedure  common.ProcedureType

	// SimUe writes messages to Profile routine on this channel
	WriteProfileChan chan *common.ProfileMessage

	// SimUe writes messages to RealUE on this channel
	WriteRealUeChan chan *common.UuMessage

	// SimUe writes messages to GnbUE on this channel
	WriteGnbUeChan chan common.InterfaceMessage

	// SimUe reads messages from other entities on this channel
	// Entities can be RealUe, GnbUe etc.
	ReadChan chan common.InterfaceMessage

	/* logger */
	Log *logrus.Entry
}

func NewSimUe(supi string, gnb *gnbctx.GNodeB, profile *profctx.Profile) *SimUe {
	simue := SimUe{}
	simue.GnB = gnb
	simue.Supi = supi
	simue.ProfileCtx = profile
	simue.ReadChan = make(chan common.InterfaceMessage)
	simue.RealUe = realuectx.NewRealUe(supi,
		security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		simue.ReadChan, profile.Plmn)
	simue.WriteRealUeChan = simue.RealUe.ReadChan
	simue.WriteProfileChan = profile.ReadChan

	simue.Log = logger.SimUeLog.WithField(logger.FieldSupi, supi)

	simue.Log.Traceln("Created new context")
	return &simue
}

//2089300007487
