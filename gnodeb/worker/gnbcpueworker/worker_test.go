// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"testing"
	"time"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/util/idgenerator"
)

// After a quit the control-plane context must be unregistered, so that the AMF cannot reach it any
// more, and its event loop must end, so that nothing is left running. Done tells the SimUe so.
func TestQuitUnregistersContextAndEndsEventLoop(t *testing.T) {
	gnb := &gnbctx.GNodeB{
		GnbUes:               gnbctx.NewGnbUeDao(),
		RanUeNGAPIDGenerator: idgenerator.NewGenerator(1, 10),
	}
	id, err := gnb.RanUeNGAPIDGenerator.Allocate()
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	gnbue := gnbctx.NewGnbCpUe(id, gnb, nil)
	gnb.GnbUes.AddGnbCpUe(id, gnbue)

	go Init(gnbue)
	gnbue.ReadChan <- &common.DefaultMessage{Event: common.QUIT_EVENT}

	select {
	case <-gnbue.Done:
	case <-time.After(2 * time.Second):
		t.Fatal("the event loop did not end after the quit")
	}
	if gnb.GnbUes.GetGnbCpUe(id) != nil {
		t.Error("the context is still registered, so AMF messages would still reach it")
	}
}

// A quit request must not depend on room in the event queue, but must not overtake the events
// that are already queued either.
func TestQuitRequestEndsEventLoopWhenQueueIsFull(t *testing.T) {
	gnb := &gnbctx.GNodeB{
		GnbUes:               gnbctx.NewGnbUeDao(),
		RanUeNGAPIDGenerator: idgenerator.NewGenerator(1, 10),
	}
	id, err := gnb.RanUeNGAPIDGenerator.Allocate()
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	gnbue := gnbctx.NewGnbCpUe(id, gnb, nil)
	gnb.GnbUes.AddGnbCpUe(id, gnbue)
	for len(gnbue.ReadChan) < cap(gnbue.ReadChan) {
		gnbue.ReadChan <- &common.DefaultMessage{Event: common.ERROR_EVENT} // not handled by the worker
	}

	gnbue.RequestQuit()
	gnbue.RequestQuit() // idempotent
	go Init(gnbue)

	select {
	case <-gnbue.Done:
	case <-time.After(2 * time.Second):
		t.Fatal("the event loop did not end after the quit request")
	}
	if len(gnbue.ReadChan) != 0 {
		t.Errorf("%d queued events were skipped", len(gnbue.ReadChan))
	}
}

// A SimUe that has stopped reading must not keep the context from terminating: the send to it
// gives way to the quit request.
func TestSendToSimUeGivesWayToQuitRequest(t *testing.T) {
	gnbue := gnbctx.NewGnbCpUe(1, &gnbctx.GNodeB{}, nil)
	gnbue.WriteUeChan = make(chan common.InterfaceMessage) // nobody reads it

	done := make(chan struct{})
	go func() {
		sendToSimUe(gnbue, &common.DefaultMessage{Event: common.DATA_BEARER_SETUP_RESULT_EVENT})
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned although the message could not be delivered and no quit was requested")
	case <-time.After(100 * time.Millisecond):
	}
	gnbue.RequestQuit()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the send is still blocked after the quit request")
	}
}

// A user plane context whose worker was never started has nobody to run its quit, so teardown must
// free its downlink tunnel ID itself, or the ID is lost for good.
func TestTerminateReleasesTeidOfContextWithoutWorker(t *testing.T) {
	gnb := &gnbctx.GNodeB{DlTeidGenerator: idgenerator.NewGenerator(1, 1)} // one ID, so a leak shows
	dlTeid, err := gnb.DlTeidGenerator.Allocate()
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	upCtx := gnbctx.NewGnbUpUe(uint32(dlTeid), 7, gnb)
	upCtx.Upf = gnbctx.NewGnbUpf("10.0.0.1")

	terminateUpUeContext(upCtx) // must not block on the command channel either

	if _, err := gnb.DlTeidGenerator.Allocate(); err != nil {
		t.Errorf("the downlink TEID was not released: %v", err)
	}
}
