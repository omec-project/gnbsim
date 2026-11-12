// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"errors"
	"testing"
	"time"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	profctx "github.com/omec-project/gnbsim/profile/context"
	simuectx "github.com/omec-project/gnbsim/simue/context"
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
	"go.uber.org/zap"
)

// testSupi is the subscriber every test in this file drives.
const testSupi = "imsi-208930100007487"

func TestHandleServiceAcceptEventReportsProcedurePass(t *testing.T) {
	profileResultChan := make(chan *common.ProfileMessage, 1)

	ue := &simuectx.SimUe{
		Supi:             testSupi,
		Procedure:        common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
		WriteProfileChan: profileResultChan,
		MsgRspReceived:   make(chan bool, 1),
		Log:              zap.NewNop().Sugar(),
		ProfileCtx:       &profctx.Profile{RetransMsg: false},
	}

	msg := &common.UeMessage{}
	msg.Event = common.SERVICE_ACCEPT_EVENT
	msg.Id = 14

	if err := HandleServiceAcceptEvent(ue, msg); err != nil {
		t.Fatalf("HandleServiceAcceptEvent returned error: %v", err)
	}

	select {
	case result := <-profileResultChan:
		if result.Event != common.PROC_PASS_EVENT {
			t.Fatalf("expected %v, got %v", common.PROC_PASS_EVENT, result.Event)
		}
		if result.Proc != common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE {
			t.Fatalf("expected procedure %v, got %v", common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE, result.Proc)
		}
		if result.Supi != ue.Supi {
			t.Fatalf("expected supi %q, got %q", ue.Supi, result.Supi)
		}
	default:
		t.Fatal("expected procedure pass message")
	}
}

// TestProcedureWithNoHandlerFailsImmediately covers the configuration mistake the default branch
// exists to surface: a procedure registered in procedures.go with no case in HandleProcedure.
//
// Logging alone is not enough. The profile is sitting in a select waiting for a result, so a
// procedure that starts and sends nothing is indistinguishable from a network that never answered
// -- the run waits out perUserTimeout and then reports "profile timeout", which is the wrong
// explanation arriving a minute late. Failing here names the cause while the UE is still on it.
//
// Reporting the failure is only half of it: the UE's workers and its bearer at the gNB outlive the
// procedure unless the same QUIT that ends any other failed procedure is sent, so both halves are
// checked here.
func TestProcedureWithNoHandlerFailsImmediately(t *testing.T) {
	profileChan := make(chan *common.ProfileMessage, 1)
	realUeChan := make(chan common.InterfaceMessage, 1)
	gnbUeChan := make(chan common.InterfaceMessage, 1)
	ue := &simuectx.SimUe{
		Supi:             testSupi,
		Log:              zap.NewNop().Sugar(),
		WriteProfileChan: profileChan,
		WriteRealUeChan:  realUeChan,
		WriteGnbUeChan:   gnbUeChan,
		// CUSTOM_PROCEDURE stands in for any procedure added to procedures.go without a case
		// in HandleProcedure. It has none, and is not one a profile runs directly.
		Procedure: common.CUSTOM_PROCEDURE,
	}

	HandleProcedure(ue)

	select {
	case msg := <-profileChan:
		if msg.Event != common.PROC_FAIL_EVENT {
			t.Fatalf("profile was told %v, want %v", msg.Event, common.PROC_FAIL_EVENT)
		}
		if msg.Error == nil {
			t.Error("the failure carries no error, so the run cannot say what went wrong")
		}
	default:
		t.Fatal("nothing was reported to the profile: the procedure fails by timeout, which is the silence this branch exists to end")
	}

	select {
	case msg := <-realUeChan:
		if msg.GetEventType() != common.QUIT_EVENT {
			t.Errorf("RealUe was told %v, want %v", msg.GetEventType(), common.QUIT_EVENT)
		}
	default:
		t.Error("the RealUe was never told to quit, so its worker outlives the procedure")
	}

	select {
	case msg := <-gnbUeChan:
		if msg.GetEventType() != common.QUIT_EVENT {
			t.Errorf("gNB UE was told %v, want %v", msg.GetEventType(), common.QUIT_EVENT)
		}
	default:
		t.Error("the gNB UE was never told to quit, so the bearer stays registered")
	}
}

// TestQuitTwiceDoesNotBlock covers the hazard the line above creates: HandleQuitEvent nils
// WriteRealUeChan after sending on it, so a second quit for the same UE would send on a nil
// channel and block that goroutine for good, with no log to say why. Nothing reaches it twice
// today, which is what makes it worth pinning rather than leaving to be discovered.
func TestQuitTwiceDoesNotBlock(t *testing.T) {
	ue := &simuectx.SimUe{
		Supi:            testSupi,
		Log:             zap.NewNop().Sugar(),
		WriteRealUeChan: make(chan common.InterfaceMessage, 2),
	}

	msg := &common.UuMessage{}
	msg.Event = common.QUIT_EVENT

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := HandleQuitEvent(ue, msg); err != nil {
			t.Errorf("the first quit returned: %v", err)
		}
		if err := HandleQuitEvent(ue, msg); err != nil {
			t.Errorf("the second quit returned: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the second quit blocked: a nil channel send never returns")
	}
}

// A session the RealUe could not establish must fail the procedure, not pass it, and the failure
// must go through the event loop so that the UE is torn down, including any session that was
// established.
func TestDataBearerSetupResponseWithFailedSessionFailsAndTearsDownUe(t *testing.T) {
	profileResultChan := make(chan *common.ProfileMessage, 2)
	gnbChan := make(chan common.InterfaceMessage, 2)
	realUeChan := make(chan common.InterfaceMessage, 2)

	ue := &simuectx.SimUe{
		Supi:             testSupi,
		WriteProfileChan: profileResultChan,
		WriteGnbUeChan:   gnbChan,
		WriteRealUeChan:  realUeChan,
		ReadChan:         make(chan common.InterfaceMessage, 1),
		Log:              zap.NewNop().Sugar(),
	}

	msg := &common.UuMessage{}
	msg.Event = common.DATA_BEARER_SETUP_RESPONSE_EVENT
	msg.DBParams = []*common.DataBearerParams{
		{PduSess: &ngapTestpacket.PduSession{PduSessId: 1, Success: true}},
		{PduSess: &ngapTestpacket.PduSession{PduSessId: 2, Success: false}},
	}
	ue.ReadChan <- msg

	// HandleEvents returns once the error has been handled
	HandleEvents(ue)

	if len(profileResultChan) != 1 {
		t.Fatalf("profile got %d results, want exactly one", len(profileResultChan))
	}
	if got := <-profileResultChan; got.Event != common.PROC_FAIL_EVENT {
		t.Errorf("profile was told %v, want %v", got.Event, common.PROC_FAIL_EVENT)
	}

	// the gNB first gets the response, so that the AMF learns of the failed session, then the quit
	if len(gnbChan) != 2 {
		t.Fatalf("gNB got %d messages, want the response followed by a quit", len(gnbChan))
	}
	if got := <-gnbChan; got.GetEventType() != common.DATA_BEARER_SETUP_RESPONSE_EVENT {
		t.Errorf("first message to the gNB was %v", got.GetEventType())
	}
	if got := <-gnbChan; got.GetEventType() != common.QUIT_EVENT {
		t.Errorf("second message to the gNB was %v, want %v", got.GetEventType(), common.QUIT_EVENT)
	}
	if len(realUeChan) != 1 || (<-realUeChan).GetEventType() != common.QUIT_EVENT {
		t.Error("the RealUe was not told to quit")
	}
}

// A profile that has given up on the UE (timeout) stops reading its result channel. The SimUe
// must still be able to tear down, and must free its SUPI only once it has done so.
func TestQuitRequestUnblocksSimUeWaitingForProfile(t *testing.T) {
	const supi = "imsi-208930100009999"
	if !simuectx.ReserveSupi(supi) {
		t.Fatal("could not reserve the SUPI")
	}
	gnbChan := make(chan common.InterfaceMessage, 2)
	realUeChan := make(chan common.InterfaceMessage, 2)
	ue := &simuectx.SimUe{
		Supi:             supi,
		WriteProfileChan: make(chan *common.ProfileMessage), // nobody reads it
		WriteGnbUeChan:   gnbChan,
		WriteRealUeChan:  realUeChan,
		ReadChan:         make(chan common.InterfaceMessage, 1),
		Log:              zap.NewNop().Sugar(),
	}
	msg := &common.UeMessage{}
	msg.Event = common.DATA_PKT_GEN_SUCCESS_EVENT // makes the SimUe report a pass to the profile
	ue.ReadChan <- msg

	done := make(chan struct{})
	go func() {
		HandleEvents(ue)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond) // let the SimUe block on the profile channel
	ue.RequestQuit()
	ue.RequestQuit() // idempotent

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the SimUe is stuck, so it could not tear down")
	}
	if len(gnbChan) != 1 || len(realUeChan) != 1 {
		t.Errorf("quit reached the gNB %d times and the RealUe %d times, want once each", len(gnbChan), len(realUeChan))
	}
	if !simuectx.ReserveSupi(supi) {
		t.Error("the SUPI was not released after the SimUe terminated")
	}
	simuectx.ReleaseSupi(supi)
}

// The gNB, not the RealUe, knows whether the bearers ended up in place, so a procedure completes
// on the gNB's result and fails on its error.
func TestDataBearerSetupResultEvent(t *testing.T) {
	newUe := func() (*simuectx.SimUe, chan *common.ProfileMessage) {
		results := make(chan *common.ProfileMessage, 1)
		return &simuectx.SimUe{
			Supi:             testSupi,
			WriteProfileChan: results,
			Log:              zap.NewNop().Sugar(),
		}, results
	}

	ue, results := newUe()
	ok := &common.UuMessage{}
	ok.Event = common.DATA_BEARER_SETUP_RESULT_EVENT
	if err := HandleDataBearerSetupResultEvent(ue, ok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := <-results; got.Event != common.PROC_PASS_EVENT {
		t.Errorf("profile was told %v, want %v", got.Event, common.PROC_PASS_EVENT)
	}

	ue, results = newUe()
	bad := &common.UuMessage{}
	bad.Event = common.DATA_BEARER_SETUP_RESULT_EVENT
	bad.Error = errors.New("no user plane context")
	if err := HandleDataBearerSetupResultEvent(ue, bad); err == nil {
		t.Error("expected the gNB's error to be returned, so that the UE is torn down")
	}
	if len(results) != 0 {
		t.Error("a failed setup must not be reported as a pass")
	}

	// the RealUe's answer alone does not complete the procedure
	ue, results = newUe()
	ue.WriteGnbUeChan = make(chan common.InterfaceMessage, 1)
	resp := &common.UuMessage{}
	resp.Event = common.DATA_BEARER_SETUP_RESPONSE_EVENT
	resp.DBParams = []*common.DataBearerParams{{PduSess: &ngapTestpacket.PduSession{PduSessId: 1, Success: true}}}
	if err := HandleDataBearerSetupResponseEvent(ue, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Error("the procedure was completed before the gNB reported its result")
	}
}

// The profile runners start the next profile as soon as a UE's execution returns, so the wait for
// teardown must not end before the SimUe says it has terminated.
func TestQuitUeAndWaitWaitsForTermination(t *testing.T) {
	ue := &simuectx.SimUe{Supi: testSupi, Log: zap.NewNop().Sugar()}
	pCtx := &profctx.ProfileUeContext{
		Quit:       ue.RequestQuit,
		Terminated: ue.Terminated(),
		Log:        zap.NewNop().Sugar(),
	}

	done := make(chan struct{})
	go func() {
		QuitUeAndWait(pCtx)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned before the SimUe terminated")
	case <-time.After(100 * time.Millisecond):
	}
	select {
	case <-ue.QuitRequested():
	default:
		t.Error("the quit was not requested")
	}

	ue.MarkTerminated()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("still waiting after the SimUe terminated")
	}
}

// The SimUe tears down every gNB context that was created for it, including one that it is no
// longer connected to, and waits for each to finish before it reports itself terminated.
func TestHandleQuitEventStopsAndAwaitsRecordedGnbContexts(t *testing.T) {
	ue := &simuectx.SimUe{
		Supi: testSupi,
		Log:  zap.NewNop().Sugar(),
		// the current queue is full: the quit must not block on it
		WriteGnbUeChan: make(chan common.InterfaceMessage),
	}
	ctxs := []*gnbctx.GnbCpUe{
		gnbctx.NewGnbCpUe(1, &gnbctx.GNodeB{}, nil), // left behind by a handover
		gnbctx.NewGnbCpUe(2, &gnbctx.GNodeB{}, nil),
	}
	for _, c := range ctxs {
		ue.AddGnbContext(c)
		go func() { // stands in for the gNB worker
			<-c.QuitRequested()
			close(c.Done)
		}()
	}

	msg := &common.UuMessage{}
	msg.Event = common.QUIT_EVENT
	done := make(chan struct{})
	go func() {
		if err := HandleQuitEvent(ue, msg); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the quit did not complete")
	}
	for i, c := range ctxs {
		select {
		case <-c.Done:
		default:
			t.Errorf("returned before gNB context %d had terminated", i)
		}
	}
}

// A service request whose Initial Context Setup also restores bearers is complete only when the
// service has been accepted and the gNB has reported the bearers, in either order and exactly once.
// Stopping the UE earlier would abandon the restoration.
func TestServiceRequestWithBearersCompletesOnceBothAreDone(t *testing.T) {
	newUe := func() (*simuectx.SimUe, chan *common.ProfileMessage) {
		results := make(chan *common.ProfileMessage, 2)
		return &simuectx.SimUe{
			Supi:             testSupi,
			Procedure:        common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
			WriteProfileChan: results,
			MsgRspReceived:   make(chan bool, 1),
			Log:              zap.NewNop().Sugar(),
			ProfileCtx:       &profctx.Profile{RetransMsg: false},
		}, results
	}
	pending := &common.UuMessage{}
	pending.Event = common.BEARER_SETUP_PENDING_EVENT
	pending.NasPdus = common.NasPduList{{0x01}} // the request carries the Service Accept
	accept := &common.UeMessage{}
	accept.Event = common.SERVICE_ACCEPT_EVENT
	result := &common.UuMessage{}
	result.Event = common.DATA_BEARER_SETUP_RESULT_EVENT

	type step struct {
		run  func(ue *simuectx.SimUe) error
		name string
	}
	acceptStep := step{
		name: "service accept",
		run:  func(ue *simuectx.SimUe) error { return HandleServiceAcceptEvent(ue, accept) },
	}
	resultStep := step{
		name: "bearer setup result",
		run:  func(ue *simuectx.SimUe) error { return HandleDataBearerSetupResultEvent(ue, result) },
	}

	for name, order := range map[string][]step{
		"accept first": {acceptStep, resultStep},
		"result first": {resultStep, acceptStep},
	} {
		ue, results := newUe()
		if err := HandleBearerSetupPendingEvent(ue, pending); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for i, st := range order {
			if err := st.run(ue); err != nil {
				t.Fatalf("%s: %s: %v", name, st.name, err)
			}
			// nothing after the first half, exactly one after the second
			if len(results) != i {
				t.Fatalf("%s: after %s there are %d results, want %d", name, st.name, len(results), i)
			}
		}
		if ue.BearerRestorePending {
			t.Errorf("%s: the restoration was not cleared", name)
		}
	}

	// without bearers the NAS accept alone completes it, and the next procedure is unaffected
	ue, results := newUe()
	if err := HandleServiceAcceptEvent(ue, accept); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results for a service request without bearers, want 1", len(results))
	}
}

// The RealUe cannot give up on a send to the SimUe, and the SimUe is the only reader. A RealUe that
// is blocked on a full queue when the SimUe quits must still be able to receive the quit.
func TestQuitDrainsRealUeBlockedOnFullQueue(t *testing.T) {
	ue := &simuectx.SimUe{
		Supi:            testSupi,
		Log:             zap.NewNop().Sugar(),
		ReadChan:        make(chan common.InterfaceMessage, 1),
		WriteRealUeChan: make(chan common.InterfaceMessage), // taken only once the RealUe reads
	}
	filler := &common.UeMessage{}
	filler.Event = common.ERROR_EVENT

	realUeQuit := ue.WriteRealUeChan
	ue.ReadChan <- filler  // the queue is full
	ue.WaitGrp.Go(func() { // stands in for the RealUe
		ue.ReadChan <- filler // blocks until the SimUe drains
		<-realUeQuit          // and only then reads its quit
	})

	msg := &common.UuMessage{}
	msg.Event = common.QUIT_EVENT
	done := make(chan struct{})
	go func() {
		if err := HandleQuitEvent(ue, msg); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the quit deadlocked against a RealUe blocked on the full queue")
	}
}

// Session resources in the Initial Context Setup Request of another procedure (a registration, for
// one) are completed by that procedure's own messages. The coordination must neither start for it
// nor carry over to the next procedure.
func TestBearerSetupPendingOnlyAppliesToServiceRequest(t *testing.T) {
	ue := &simuectx.SimUe{
		Supi:      testSupi,
		Procedure: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
		Log:       zap.NewNop().Sugar(),
		// where the registration request that starting the procedure sends goes
		WriteRealUeChan: make(chan common.InterfaceMessage, 1),
	}
	pending := &common.UuMessage{}
	pending.Event = common.BEARER_SETUP_PENDING_EVENT
	if err := HandleBearerSetupPendingEvent(ue, pending); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ue.BearerRestorePending {
		t.Error("a procedure that has no NAS half to wait for was made to wait")
	}

	// left over from a service request that was cut short
	ue.BearerRestorePending, ue.ServiceAccepted, ue.BearersRestored = true, true, true
	start := &common.ProfileMessage{}
	start.Event = common.PROC_START_EVENT
	start.Proc = common.REGISTRATION_PROCEDURE
	if err := HandleProcedureEvent(ue, start); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ue.BearerRestorePending || ue.ServiceAccepted || ue.BearersRestored {
		t.Error("state of the previous procedure was carried into the next one")
	}
}

// The ordinary sends of the SimUe must give way to a quit request, or a full queue keeps the event
// loop from ever reaching its teardown.
func TestSendsGiveWayToQuitRequest(t *testing.T) {
	ue := &simuectx.SimUe{
		Supi:            testSupi,
		Log:             zap.NewNop().Sugar(),
		WriteRealUeChan: make(chan common.InterfaceMessage), // nobody reads these
		WriteGnbUeChan:  make(chan common.InterfaceMessage),
	}
	msg := &common.UeMessage{}
	msg.Event = common.REG_REQUEST_EVENT

	done := make(chan struct{})
	go func() {
		SendToRealUe(ue, msg)
		SendToGnbUe(ue, msg)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned although nothing could be delivered and no quit was requested")
	case <-time.After(100 * time.Millisecond):
	}
	ue.RequestQuit()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a send is still blocked after the quit request")
	}
}

// A service request whose Initial Context Setup carries no NAS-PDU is completed by the restored
// bearers alone, and a Service Accept that comes separately afterwards does not complete it twice.
func TestServiceRequestWithoutNasPduCompletesOnBearers(t *testing.T) {
	results := make(chan *common.ProfileMessage, 2)
	ue := &simuectx.SimUe{
		Supi:             testSupi,
		Procedure:        common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
		WriteProfileChan: results,
		MsgRspReceived:   make(chan bool, 1),
		Log:              zap.NewNop().Sugar(),
		ProfileCtx:       &profctx.Profile{RetransMsg: false},
	}
	pending := &common.UuMessage{} // no NasPdus: the request does not carry the Service Accept
	pending.Event = common.BEARER_SETUP_PENDING_EVENT
	if err := HandleBearerSetupPendingEvent(ue, pending); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := &common.UuMessage{}
	result.Event = common.DATA_BEARER_SETUP_RESULT_EVENT
	if err := HandleDataBearerSetupResultEvent(ue, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results from the bearers alone, want 1", len(results))
	}

	accept := &common.UeMessage{}
	accept.Event = common.SERVICE_ACCEPT_EVENT
	if err := HandleServiceAcceptEvent(ue, accept); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("a late Service Accept completed the procedure a second time (%d results)", len(results))
	}
}

// A Service Accept can come after the bearers completed the service request and the next
// procedure has started. It is still the late answer to the old request: it must neither fail the
// new procedure nor be taken for the answer to a later service request.
func TestLateServiceAcceptSurvivesNextProcedure(t *testing.T) {
	results := make(chan *common.ProfileMessage, 2)
	ue := &simuectx.SimUe{
		Supi:             testSupi,
		Procedure:        common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
		WriteProfileChan: results,
		WriteRealUeChan:  make(chan common.InterfaceMessage, 2),
		Log:              zap.NewNop().Sugar(),
		ProfileCtx:       &profctx.Profile{RetransMsg: false},
	}
	ue.ServiceCompleted = true // the bearers completed the service request

	start := &common.ProfileMessage{}
	start.Event = common.PROC_START_EVENT
	start.Proc = common.REGISTRATION_PROCEDURE // any procedure that is not a service request
	if err := HandleProcedureEvent(ue, start); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ue.ServiceCompleted {
		t.Fatal("the late Accept was forgotten when the next procedure started")
	}

	accept := &common.UeMessage{}
	accept.Event = common.SERVICE_ACCEPT_EVENT
	if err := HandleServiceAcceptEvent(ue, accept); err != nil {
		t.Errorf("the late Accept failed the procedure that was running: %v", err)
	}
	if len(results) != 0 {
		t.Error("the late Accept was reported to the profile")
	}

	// a new service request does not inherit it: its own Accept must be handled
	ue.ServiceCompleted = true
	start.Proc = common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE
	if err := HandleProcedureEvent(ue, start); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ue.ServiceCompleted {
		t.Error("a new service request inherited the late Accept of the previous one")
	}
}

// The worker that resends a Service Request ends when its request is answered or completed, when
// the UE quits, and when it has nothing left to resend; it does not wait for a shared signal.
func TestRetransmitMsgEnds(t *testing.T) {
	newUe := func() *simuectx.SimUe {
		return &simuectx.SimUe{
			Supi:           testSupi,
			Log:            zap.NewNop().Sugar(),
			WriteGnbUeChan: make(chan common.InterfaceMessage, 4),
		}
	}
	msg := &common.UeMessage{}
	msg.Event = common.SERVICE_REQUEST_EVENT

	wait := func(name string, done <-chan struct{}) {
		t.Helper()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: the worker did not end", name)
		}
	}

	ue := newUe()
	startRetransmission(ue, msg, 2)
	stop := ue.RetransmitStop
	if stop == nil {
		t.Fatal("no stop channel for the worker")
	}
	stopRetransmission(ue)
	stopRetransmission(ue) // idempotent
	if ue.RetransmitStop != nil {
		t.Error("the stop channel was kept")
	}
	select {
	case <-stop:
	default:
		t.Error("the worker was not told to stop")
	}

	ue = newUe()
	done := make(chan struct{})
	go func() {
		retransmitMsg(ue, msg, 2, retransmitInterval, make(chan struct{}))
		close(done)
	}()
	ue.RequestQuit()
	wait("quit", done)
}

// A resend that is blocked on a full queue must not go out once its request has been answered: it
// would otherwise land during the next procedure.
func TestRetransmitMsgBlockedSendGivesWayToStop(t *testing.T) {
	full := make(chan common.InterfaceMessage) // nobody reads it, so a send blocks
	ue := &simuectx.SimUe{Supi: testSupi, Log: zap.NewNop().Sugar(), WriteGnbUeChan: full}
	msg := &common.UeMessage{}
	msg.Event = common.SERVICE_REQUEST_EVENT

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		retransmitMsg(ue, msg, 2, 10*time.Millisecond, stop)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond) // the ticker fires and the worker blocks in its send
	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the worker stayed blocked in its send after its request was answered")
	}
}

// A registration whose Initial Context Setup also restores bearers is done only when the
// Registration Complete is sent and the gNB has reported the bearers, in either order.
func TestRegistrationWithBearersCompletesOnceBothAreDone(t *testing.T) {
	for name, regCompleteFirst := range map[string]bool{"complete first": true, "bearers first": false} {
		results := make(chan *common.ProfileMessage, 2)
		ue := &simuectx.SimUe{
			Supi:             testSupi,
			Procedure:        common.REGISTRATION_PROCEDURE,
			WriteProfileChan: results,
			WriteGnbUeChan:   make(chan common.InterfaceMessage, 2),
			Log:              zap.NewNop().Sugar(),
			ProfileCtx:       &profctx.Profile{},
		}
		pending := &common.UuMessage{}
		pending.Event = common.BEARER_SETUP_PENDING_EVENT
		if err := HandleBearerSetupPendingEvent(ue, pending); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		regComplete := func() {
			msg := &common.UuMessage{}
			msg.Event = common.REG_COMPLETE_EVENT
			if err := HandleRegCompleteEvent(ue, msg); err != nil {
				t.Fatalf("%s: registration complete: %v", name, err)
			}
		}
		bearers := func() {
			result := &common.UuMessage{}
			result.Event = common.DATA_BEARER_SETUP_RESULT_EVENT
			if err := HandleDataBearerSetupResultEvent(ue, result); err != nil {
				t.Fatalf("%s: bearer result: %v", name, err)
			}
		}
		first, second := bearers, regComplete
		if regCompleteFirst {
			first, second = regComplete, bearers
		}
		first()
		if len(results) != 0 {
			t.Fatalf("%s: the registration was reported before both halves were done", name)
		}
		second()
		if len(results) != 1 {
			t.Fatalf("%s: got %d results, want exactly one", name, len(results))
		}
	}
}
