// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package pdusessworker

import (
	"net"
	"testing"
	"time"

	"github.com/omec-project/gnbsim/common"
	realuectx "github.com/omec-project/gnbsim/realue/context"
	"go.uber.org/zap"
)

// addresses that the tests give a session: its own, and where its packets are headed
const (
	testUeAddr    = "10.0.0.1"
	testDefaultAs = "192.168.250.1"
)

// Moving to the target gNB at handover must tell the source gNB's user plane worker that no more
// uplink packets follow, or that worker can never terminate.
func TestHandleInitEventTellsPreviousGnbTheLastPacket(t *testing.T) {
	source := make(chan common.InterfaceMessage, 1)
	target := make(chan common.InterfaceMessage, 1)
	pduSess := &realuectx.PduSession{WriteGnbChan: source, Log: zap.NewNop().Sugar()}

	init := &common.UeMessage{}
	init.Event = common.INIT_EVENT
	init.CommChan = target
	if err := HandleInitEvent(pduSess, init); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case got := <-source:
		if got.GetEventType() != common.LAST_DATA_PKT_EVENT {
			t.Errorf("the previous gNB was sent %v", got.GetEventType())
		}
	default:
		t.Error("the previous gNB was not told that the last uplink packet has been sent")
	}
	if pduSess.WriteGnbChan != target {
		t.Error("the session was not moved to the new gNB")
	}

	// the first setup has no previous gNB, and re-initialising towards the same one is not a move
	pduSess = &realuectx.PduSession{Log: zap.NewNop().Sugar()}
	if err := HandleInitEvent(pduSess, init); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := HandleInitEvent(pduSess, init); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(target) != 0 {
		t.Error("a marker was sent to a gNB that was not left")
	}
}

// The quit must neither wait for a gNB that is not reading nor leave the packet generator running
// against the channels it clears.
func TestHandleQuitEventIsBoundedAndStopsGenerator(t *testing.T) {
	old := lastPktWaitTimeout
	lastPktWaitTimeout = 100 * time.Millisecond
	defer func() { lastPktWaitTimeout = old }()

	// nobody reads it, and the generator's first packet fills it, so the marker cannot be sent
	pduSess := &realuectx.PduSession{
		WriteGnbChan: make(chan common.InterfaceMessage, 1),
		ReadDlChan:   make(chan common.InterfaceMessage, 10),
		WriteUeChan:  make(chan common.InterfaceMessage, 10),
		PduAddress:   net.ParseIP(testUeAddr),
		Log:          zap.NewNop().Sugar(),
	}

	// a generator that is asleep between two packets, and so cannot notice the quit by itself
	gen := &common.UeMessage{}
	gen.UserDataPktCount = 5
	gen.UserDataPktInterval = 3600
	gen.DefaultAs = testDefaultAs
	if err := HandleDataPktGenRequestEvent(pduSess, gen); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	done := make(chan struct{})
	go func() {
		if err := HandleQuitEvent(pduSess, &common.UeMessage{}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the quit did not return although the gNB is not reading and no marker will come")
	}
	if pduSess.GenStop != nil {
		t.Error("the generator was not stopped")
	}
	if pduSess.WriteGnbChan != nil {
		t.Error("the channel to the gNB was not released")
	}
}

// After an AN release the session has no gNB. A request to send data must fail at once: waiting on
// a nil channel would keep the worker from ever processing its quit.
func TestSendIcmpEchoRequestRefusesDisconnectedSession(t *testing.T) {
	pduSess := &realuectx.PduSession{
		PduAddress: net.ParseIP(testUeAddr),
		DefaultAs:  testDefaultAs,
		Log:        zap.NewNop().Sugar(),
	}
	done := make(chan error, 1)
	go func() { done <- SendIcmpEchoRequest(pduSess) }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a send on a disconnected session reported success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the send blocked on a session that has no gNB")
	}
}

// A one-off packet runs on the session's event loop. Blocked on a gNB that is not reading, it must
// give up, or the session could never process its quit.
func TestSendIcmpEchoRequestIsBoundedWhenGnbIsNotReading(t *testing.T) {
	old := ulSendTimeout
	ulSendTimeout = 50 * time.Millisecond
	defer func() { ulSendTimeout = old }()

	pduSess := &realuectx.PduSession{
		WriteGnbChan: make(chan common.InterfaceMessage), // nobody reads it
		PduAddress:   net.ParseIP(testUeAddr),
		DefaultAs:    testDefaultAs,
		Log:          zap.NewNop().Sugar(),
	}
	done := make(chan error, 1)
	go func() { done <- SendIcmpEchoRequest(pduSess) }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a send that could not be delivered reported success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the send stayed blocked on a gNB that is not reading")
	}
}
