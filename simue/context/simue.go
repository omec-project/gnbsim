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
	"github.com/omec-project/nas/v2/security"
	"go.uber.org/zap"
)

func init() {
	simUeTable = make(map[string]*SimUe)
}

// SimUe controls the flow of messages between RealUe and GnbUe as per the test
// profile. It is the central entry point for all events
type SimUe struct {
	TargetGnB            *gnbctx.GNodeB
	RealUe               *realuectx.RealUe
	ProfileCtx           *profctx.Profile
	ReadChan             chan common.InterfaceMessage
	WriteProfileChan     chan *common.ProfileMessage
	WriteRealUeChan      chan common.InterfaceMessage
	MsgRspReceived       chan bool
	GnB                  *gnbctx.GNodeB
	TargetWriteGnbUeChan chan common.InterfaceMessage
	RetransmitStop       chan struct{}
	WriteGnbUeChan       chan common.InterfaceMessage
	terminated           chan struct{}
	quit                 chan struct{}
	Log                  *zap.SugaredLogger
	Supi                 string
	gnbCtxs              []*gnbctx.GnbCpUe
	WaitGrp              sync.WaitGroup
	quitMu               sync.Mutex
	gnbCtxMu             sync.Mutex
	AcceptExpected       bool
	terminatedClosed     bool
	BearerRestorePending bool
	ServiceAccepted      bool
	BearersRestored      bool
	quitClosed           bool
	ServiceCompleted     bool
	Procedure            common.ProcedureType
	// PassedByAccept is set when a Service Accept completed the service request on its own, before
	// the gNB had said that bearers are being restored. If that notice comes afterwards the
	// procedure cannot be undone, so IgnoreBearerResult makes the bearer result that follows it
	// count for nothing, instead of being reported as a second pass.
	//
	// Both name the gNB context of the request (its RAN UE NGAP ID), so that they apply to that
	// request's notice and result only, and stay in force across the start of other procedures.
	PassedByAccept     bool
	IgnoreBearerResult bool
	PassedByAcceptId   int64
	IgnoreResultId     int64

	// RetransmitWg tracks the worker that resends the Service Request, so that stopping it can
	// wait until it has returned and cannot enqueue a stale request afterwards.
	RetransmitWg sync.WaitGroup
}

// AddGnbContext records a gNB control-plane context that was created for this UE. The SimUe keeps
// every one of them, whatever happens to its connection, because it has to tear them all down.
func (s *SimUe) AddGnbContext(c *gnbctx.GnbCpUe) {
	if c == nil {
		return
	}
	s.gnbCtxMu.Lock()
	s.gnbCtxs = append(s.gnbCtxs, c)
	s.gnbCtxMu.Unlock()
}

// GnbContexts returns the gNB control-plane contexts created for this UE: the one it connected to
// and any handover target, including those it has since left behind.
func (s *SimUe) GnbContexts() []*gnbctx.GnbCpUe {
	s.gnbCtxMu.Lock()
	defer s.gnbCtxMu.Unlock()
	return append([]*gnbctx.GnbCpUe(nil), s.gnbCtxs...)
}

// Terminated returns a channel that is closed once the SimUe has finished tearing down, which
// includes its RealUe and the gNB contexts that served it.
func (s *SimUe) Terminated() <-chan struct{} {
	s.quitMu.Lock()
	defer s.quitMu.Unlock()
	if s.terminated == nil {
		s.terminated = make(chan struct{})
	}
	return s.terminated
}

// MarkTerminated closes the channel returned by Terminated. It is idempotent.
func (s *SimUe) MarkTerminated() {
	s.quitMu.Lock()
	defer s.quitMu.Unlock()
	if s.terminated == nil {
		s.terminated = make(chan struct{})
	}
	if !s.terminatedClosed {
		s.terminatedClosed = true
		close(s.terminated)
	}
}

// QuitRequested returns a channel that is closed once RequestQuit has been called.
func (s *SimUe) QuitRequested() <-chan struct{} {
	s.quitMu.Lock()
	defer s.quitMu.Unlock()
	if s.quit == nil {
		s.quit = make(chan struct{})
	}
	return s.quit
}

// RequestQuit asks the SimUe to tear down. Unlike an event on ReadChan it cannot be dropped when
// the queue is full, and it also wakes a SimUe that is blocked delivering a result to a profile
// that has stopped listening. It is idempotent.
func (s *SimUe) RequestQuit() {
	s.quitMu.Lock()
	defer s.quitMu.Unlock()
	if s.quit == nil {
		s.quit = make(chan struct{})
	}
	if !s.quitClosed {
		s.quitClosed = true
		close(s.quit)
	}
}

var (
	simUeTable      map[string]*SimUe
	simUeTableMutex sync.RWMutex

	// activeSupis holds the SUPIs of the UEs that were created for a profile and have not
	// terminated yet. SimUe lookups are keyed by SUPI, so two profiles running the same SUPI
	// would route each other's procedures and results, and two live UEs with one SUPI would
	// compete for the same NGAP context.
	activeSupis      = make(map[string]struct{})
	activeSupisMutex sync.Mutex
)

// ReserveSupi marks the SUPI as in use by a running profile. It reports false, and reserves
// nothing, if the SUPI is already reserved.
func ReserveSupi(supi string) bool {
	activeSupisMutex.Lock()
	defer activeSupisMutex.Unlock()
	if _, busy := activeSupis[supi]; busy {
		return false
	}
	activeSupis[supi] = struct{}{}
	return true
}

// ReleaseSupi releases a reservation made with ReserveSupi.
func ReleaseSupi(supi string) {
	activeSupisMutex.Lock()
	delete(activeSupis, supi)
	activeSupisMutex.Unlock()
}

// DeleteSimUe removes the SimUe from the table, unless a newer one has replaced it.
func DeleteSimUe(simue *SimUe) {
	simUeTableMutex.Lock()
	defer simUeTableMutex.Unlock()
	if simUeTable[simue.Supi] == simue {
		delete(simUeTable, simue.Supi)
	}
}

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
	simue.RealUe.ModificationRequestType = profile.ModificationRequestType
	simue.RealUe.OmitModificationRequestType = profile.OmitModificationRequestType
	simue.WriteRealUeChan = simue.RealUe.ReadChan
	simue.WriteProfileChan = result

	simue.Log = logger.SimUeLog.With(logger.FieldSupi, supi)

	simue.Log.Debugln("created new SimUe context")
	simue.MsgRspReceived = make(chan bool, 5)
	simUeTableMutex.Lock()
	defer simUeTableMutex.Unlock()
	simUeTable[supi] = &simue
	return &simue
}

func GetSimUe(supi string) *SimUe {
	simUeTableMutex.RLock()
	defer simUeTableMutex.RUnlock()
	simue, found := simUeTable[supi]
	if !found {
		return nil
	}
	return simue
}
