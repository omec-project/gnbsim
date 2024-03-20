// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/logger"
	profctx "github.com/omec-project/gnbsim/profile/context"
	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/nas/security"
	"github.com/sirupsen/logrus"
)

func init() {
	SimUeTable = make(map[string]*SimUe)
}

// SimUe controls the flow of messages between RealUe and GnbUe as per the test
// profile. It is the central entry point for all events
type SimUe struct {
	GnB        *gnbctx.GNodeB
	RealUe     *realuectx.RealUe
	ProfileCtx *profctx.Profile
	Log        *logrus.Entry

	// SimUe writes messages to Profile routine on this channel
	WriteProfileChan chan *common.ProfileMessage

	// SimUe writes messages to RealUE on this channel
	WriteRealUeChan chan common.InterfaceMessage

	// SimUe writes messages to GnbUE on this channel
	WriteGnbUeChan chan common.InterfaceMessage

	// SimUe reads messages from other entities on this channel
	// Entities can be RealUe, GnbUe etc.
	ReadChan chan common.InterfaceMessage

	// Message response received
	MsgRspReceived chan bool

	Supi      string
	Procedure common.ProcedureType
	WaitGrp   sync.WaitGroup
}

var SimUeTable map[string]*SimUe

func NewSimUe(supi string, gnb *gnbctx.GNodeB, profile *profctx.Profile, result chan *common.ProfileMessage) *SimUe {
	simue := SimUe{}
	simue.GnB = gnb
	simue.Supi = supi
	simue.ProfileCtx = profile
	simue.ReadChan = make(chan common.InterfaceMessage, 5)
	simue.RealUe = realuectx.NewRealUe(supi,
		security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		simue.ReadChan, profile.Plmn, profile.Key, profile.Opc, profile.SeqNum,
		profile.Dnn, profile.SNssai)
	simue.WriteRealUeChan = simue.RealUe.ReadChan
	simue.WriteProfileChan = result

	simue.Log = logger.SimUeLog.WithField(logger.FieldSupi, supi)

	simue.Log.Traceln("Created new SimUe context")
	simue.MsgRspReceived = make(chan bool, 5)
	SimUeTable[supi] = &simue
	return &simue
}

func GetSimUe(supi string) *SimUe {
	simue, found := SimUeTable[supi]
	if !found {
		return nil
	}
	return simue
}
