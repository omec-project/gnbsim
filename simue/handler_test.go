// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"testing"
	"time"

	"github.com/omec-project/gnbsim/common"
	profctx "github.com/omec-project/gnbsim/profile/context"
	simuectx "github.com/omec-project/gnbsim/simue/context"
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
