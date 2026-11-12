// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"go.uber.org/zap"
)

type GnbCpUe struct {
	quit        chan struct{}
	Gnb         *GNodeB
	Log         *zap.SugaredLogger
	WriteUeChan chan common.InterfaceMessage
	ReadChan    chan common.InterfaceMessage
	Done        chan struct{}
	Amf         *GnbAmf
	GnbUpUes    sync.Map
	Supi        string
	WaitGrp     sync.WaitGroup
	GnbUeNgapId int64
	AmfUeNgapId int64
	quitMu      sync.Mutex
	quitClosed  bool

	// HandoverNonce tells apart the registrations of a pre-registered handover target that happen
	// to get the same RAN UE NGAP ID one after the other. It is part of the token that names the
	// target in a HandoverRequest, and 0 for a context that is not a handover target.
	HandoverNonce uint64
}

// QuitRequested returns a channel that is closed once RequestQuit has been called.
func (ctx *GnbCpUe) QuitRequested() <-chan struct{} {
	ctx.quitMu.Lock()
	defer ctx.quitMu.Unlock()
	if ctx.quit == nil {
		ctx.quit = make(chan struct{})
	}
	return ctx.quit
}

// RequestQuit asks the event loop of this context to tear it down once the events already queued
// have been handled. Unlike a QUIT_EVENT on ReadChan it cannot be dropped when the queue is full.
// It is idempotent.
func (ctx *GnbCpUe) RequestQuit() {
	ctx.quitMu.Lock()
	defer ctx.quitMu.Unlock()
	if ctx.quit == nil {
		ctx.quit = make(chan struct{})
	}
	if !ctx.quitClosed {
		ctx.quitClosed = true
		close(ctx.quit)
	}
}

func NewGnbCpUe(ngapId int64, gnb *GNodeB, amf *GnbAmf) *GnbCpUe {
	gnbue := GnbCpUe{}
	gnbue.GnbUeNgapId = ngapId
	gnbue.Amf = amf
	gnbue.Gnb = gnb
	gnbue.ReadChan = make(chan common.InterfaceMessage, 5)
	gnbue.Done = make(chan struct{})
	gnbue.Log = logger.GNodeBLog.With("subcategory", "GnbCpUe", logger.FieldGnbUeNgapId, ngapId)
	gnbue.Log.Debugln("context created")
	return &gnbue
}

// GetGnbUpUe returns the GnbUpUe instance corresponding to provided PDU Sess ID
func (ctx *GnbCpUe) GetGnbUpUe(pduSessId int64) (*GnbUpUe, error) {
	ctx.Log.Infoln("fetching GnbUpUe for pduSessId:", pduSessId)
	val, ok := ctx.GnbUpUes.Load(pduSessId)
	if ok {
		return val.(*GnbUpUe), nil
	} else {
		return nil, fmt.Errorf("key not present: %v", pduSessId)
	}
}

// AddGnbUpUe adds the GnbUpUe instance corresponding to provided PDU Sess ID
func (ctx *GnbCpUe) AddGnbUpUe(pduSessId int64, gnbue *GnbUpUe) {
	ctx.Log.Infoln("adding new GnbUpUe for PDU Sess ID:", pduSessId)
	ctx.GnbUpUes.Store(pduSessId, gnbue)
}

// RemoveGnbUpUe removes the GnbUpUe instance corresponding to provided PDU
// sess ID from the map
func (ctx *GnbCpUe) RemoveGnbUpUe(pduSessId int64) {
	ctx.Log.Infoln("deleting GnbUpUe for pduSessId:", pduSessId)
	ctx.GnbUpUes.Delete(pduSessId)
}
