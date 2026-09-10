// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package realue

import (
	"testing"

	"github.com/omec-project/gnbsim/common"
	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"go.uber.org/zap"
)

// TestResolveModificationRequestType drives the decision the UE makes about the Request type IE,
// rather than the builder that receives it.
//
// The builder tests in realue/nas cover what each value produces, and they pass a value in
// directly -- so none of them can fail for a caller that never produces the value. The case that
// matters is the one those tests cannot see: a profile setting modificationRequestType and
// omitModificationRequestType together, which is how they read sitting side by side in the config.
func TestResolveModificationRequestType(t *testing.T) {
	const omitted = 0

	tests := []struct {
		name       string
		configured uint8
		omit       bool
		want       uint8
	}{
		{
			name: "unset means modification request",
			want: nasMessage.ULNASTransportRequestTypeModificationRequest,
		},
		{
			name:       "a configured value is kept",
			configured: nasMessage.ULNASTransportRequestTypeInitialRequest,
			want:       nasMessage.ULNASTransportRequestTypeInitialRequest,
		},
		{
			name: "omit with nothing configured leaves the IE out",
			omit: true,
			want: omitted,
		},
		{
			name:       "omit wins over a configured value",
			configured: nasMessage.ULNASTransportRequestTypeModificationRequest,
			omit:       true,
			want:       omitted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveModificationRequestType(tt.configured, tt.omit); got != tt.want {
				t.Errorf("resolveModificationRequestType(%d, %v) = %d, want %d",
					tt.configured, tt.omit, got, tt.want)
			}
		})
	}
}

// newRejectMessage builds the decoded DL message the RealUe would hand on, carrying a
// PDU Session Modification Reject for the given session and PTI.
func newRejectMessage(pduSessID, pti uint8) *common.UeMessage {
	nasMsg := nas.NewMessage()
	nasMsg.GsmMessage = nas.NewGsmMessage()
	nasMsg.PDUSessionModificationReject = nasMessage.NewPDUSessionModificationReject(nas.MsgTypePDUSessionModificationReject)
	nasMsg.PDUSessionModificationReject.PDUSessionID.Octet = pduSessID
	nasMsg.PDUSessionModificationReject.PTI.Octet = pti
	nasMsg.PDUSessionModificationReject.Cause5GSM.Octet = 0x2b

	m := &common.UeMessage{}
	m.Event = common.PDU_SESS_MOD_REJECT_EVENT
	m.NasMsg = nasMsg
	return m
}

func newUeWithOutstandingRequest(pduSessID int64, outstandingPTI uint8) (*realuectx.RealUe, chan common.InterfaceMessage) {
	simUeChan := make(chan common.InterfaceMessage, 1)
	ue := &realuectx.RealUe{
		Supi:           "imsi-208930100007487",
		Log:            zap.NewNop().Sugar(),
		WriteSimUeChan: simUeChan,
		PduSessions:    make(map[int64]*realuectx.PduSession),
	}
	ue.PduSessions[pduSessID] = &realuectx.PduSession{PendingPTI: outstandingPTI}
	return ue, simUeChan
}

// TestRejectWithWrongPtiNeverReachesSimUe is the invariant behind where the PTI check lives.
//
// The SimUe reports PROC_PASS the moment it hears about a reject, unconditionally, and it sends
// that before it can hear anything back from the RealUe -- the SimUe is a single goroutine, and
// its handler reported the verdict before returning to its event loop. So a check that runs on
// the RealUe side of that exchange cannot change the verdict: the profile has already taken the
// pass and moved to the next procedure, and the later failure is attributed to whatever ran next.
//
// Reordering forwardDlNasToSimUe to send before it checks fails this test, which is the shape of
// the defect it exists to prevent.
func TestRejectWithWrongPtiNeverReachesSimUe(t *testing.T) {
	const pduSessID = 10
	ue, simUeChan := newUeWithOutstandingRequest(pduSessID, 7)

	err := forwardDlNasToSimUe(ue, newRejectMessage(pduSessID, 9),
		nas.MsgTypePDUSessionModificationReject)
	if err == nil {
		t.Fatal("expected an error for a reject carrying a PTI the UE did not use")
	}

	select {
	case msg := <-simUeChan:
		t.Fatalf("an unmatched reject was reported to the SimUe as %v, which makes it a pass",
			msg.GetEventType())
	default:
	}

	if got := ue.PduSessions[pduSessID].PendingPTI; got != 7 {
		t.Errorf("outstanding PTI = %d, want it left at 7: nothing has answered that request", got)
	}
}

// TestRejectWithMatchingPtiReachesSimUe is the other half: a reject that does belong to the
// UE's request must be reported, or the procedure would fail by timeout instead of passing.
func TestRejectWithMatchingPtiReachesSimUe(t *testing.T) {
	const pduSessID = 10
	ue, simUeChan := newUeWithOutstandingRequest(pduSessID, 7)

	err := forwardDlNasToSimUe(ue, newRejectMessage(pduSessID, 7),
		nas.MsgTypePDUSessionModificationReject)
	if err != nil {
		t.Fatalf("forwardDlNasToSimUe returned error for a matching reject: %v", err)
	}

	select {
	case msg := <-simUeChan:
		if msg.GetEventType() != common.PDU_SESS_MOD_REJECT_EVENT {
			t.Fatalf("SimUe was told %v, want %v", msg.GetEventType(), common.PDU_SESS_MOD_REJECT_EVENT)
		}
	default:
		t.Fatal("a matching reject was not reported to the SimUe, so the procedure would time out")
	}

	if got := ue.PduSessions[pduSessID].PendingPTI; got != 0 {
		t.Errorf("outstanding PTI = %d, want 0: the request has been answered", got)
	}
}
