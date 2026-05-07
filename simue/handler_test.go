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
