// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simue

import (
	"testing"

	"github.com/omec-project/gnbsim/common"
	profctx "github.com/omec-project/gnbsim/profile/context"
	simuectx "github.com/omec-project/gnbsim/simue/context"
	"go.uber.org/zap"
)

func TestHandleServiceAcceptEventReportsProcedurePass(t *testing.T) {
	profileResultChan := make(chan *common.ProfileMessage, 1)

	ue := &simuectx.SimUe{
		Supi:             "imsi-208930100007487",
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
func TestProcedureWithNoHandlerFailsImmediately(t *testing.T) {
	profileChan := make(chan *common.ProfileMessage, 1)
	ue := &simuectx.SimUe{
		Supi:             "imsi-208930100007487",
		Log:              zap.NewNop().Sugar(),
		WriteProfileChan: profileChan,
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
}
