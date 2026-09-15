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

// pduSessID is the session the simulator establishes, and the one every case here drives.
const pduSessID = 10

func newUeWithOutstandingRequest(outstandingPTI uint8) (*realuectx.RealUe, chan common.InterfaceMessage) {
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
	ue, simUeChan := newUeWithOutstandingRequest(7)

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
	ue, simUeChan := newUeWithOutstandingRequest(7)

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

// TestRejectWithNothingOutstandingNeverReachesSimUe covers the reject that answers a transaction
// which is already over. The first reject clears the outstanding PTI, so a second one arrives
// with nothing to match -- and the old guard, which only compared PTIs when one was outstanding,
// let it through as a pass for whichever procedure the profile had moved on to.
func TestRejectWithNothingOutstandingNeverReachesSimUe(t *testing.T) {
	ue, simUeChan := newUeWithOutstandingRequest(0)

	err := forwardDlNasToSimUe(ue, newRejectMessage(pduSessID, 7),
		nas.MsgTypePDUSessionModificationReject)
	if err == nil {
		t.Fatal("expected an error for a reject arriving with no request outstanding")
	}

	select {
	case msg := <-simUeChan:
		t.Fatalf("a reject answering nothing was reported to the SimUe as %v, which makes it a pass",
			msg.GetEventType())
	default:
	}
}

// TestRejectForUnknownSessionNeverReachesSimUe covers a reject naming a PDU session this UE does
// not hold. The old guard skipped the check entirely in that case -- the answer belongs to no
// request the UE made, which is the same misattribution one place further out.
func TestRejectForUnknownSessionNeverReachesSimUe(t *testing.T) {
	ue, simUeChan := newUeWithOutstandingRequest(7)

	err := forwardDlNasToSimUe(ue, newRejectMessage(pduSessID+1, 7),
		nas.MsgTypePDUSessionModificationReject)
	if err == nil {
		t.Fatal("expected an error for a reject naming a session the UE does not hold")
	}

	select {
	case msg := <-simUeChan:
		t.Fatalf("a reject for an unknown session was reported to the SimUe as %v, which makes it a pass",
			msg.GetEventType())
	default:
	}

	if got := ue.PduSessions[pduSessID].PendingPTI; got != 7 {
		t.Errorf("outstanding PTI = %d, want it left at 7: nothing has answered that request", got)
	}
}

// TestPendingPtiIsRecordedOnlyWhenTheRequestWasSent covers what the reject guard depends on.
//
// The guard asks whether a request is outstanding on the session, so the PTI has to mean "a
// request carrying this PTI left the UE". Recorded before the message was built and encrypted, a
// failure in either left the UE holding a PTI for a request that never went anywhere -- and the
// next reject to arrive, belonging to nothing, would have been matched to it and reported as the
// network refusing correctly.
func TestPendingPtiIsRecordedOnlyWhenTheRequestWasSent(t *testing.T) {
	t.Run("the request cannot be encrypted", func(t *testing.T) {
		ue, simUeChan := newUeWithOutstandingRequest(0)
		// No such integrity algorithm, so the encryption step fails after the message is built.
		ue.IntegrityAlg = 0x0f

		if err := HandlePduSessModificationRequestEvent(ue, &common.UeMessage{}); err == nil {
			t.Fatal("expected an error when the request cannot be encrypted")
		}

		select {
		case msg := <-simUeChan:
			t.Fatalf("a request that could not be encrypted was sent on as %v", msg.GetEventType())
		default:
		}

		if got := ue.PduSessions[pduSessID].PendingPTI; got != 0 {
			t.Errorf("outstanding PTI = %d, want 0: nothing was sent, so nothing is outstanding", got)
		}
	})

	t.Run("the request is sent", func(t *testing.T) {
		ue, simUeChan := newUeWithOutstandingRequest(0)

		if err := HandlePduSessModificationRequestEvent(ue, &common.UeMessage{}); err != nil {
			t.Fatalf("HandlePduSessModificationRequestEvent returned: %v", err)
		}

		select {
		case msg := <-simUeChan:
			if msg.GetEventType() != common.PDU_SESS_MOD_REQUEST_EVENT {
				t.Fatalf("SimUe was told %v, want %v", msg.GetEventType(), common.PDU_SESS_MOD_REQUEST_EVENT)
			}
		default:
			t.Fatal("the request never reached the SimUe")
		}

		if got := ue.PduSessions[pduSessID].PendingPTI; got != modificationRequestPTI {
			t.Errorf("outstanding PTI = %d, want %d: the answer has to be matchable to this request",
				got, modificationRequestPTI)
		}
	})
}

// The collision case, from the UE's side. TS 24.501 subclause 6.3.2.6 b): a modification command
// arriving during the UE's own procedure, with no procedure transaction identity assigned and
// naming the session the UE asked about, has the UE abort its own procedure internally.
//
// Aborting it means forgetting the transaction, and that is the half worth testing: left
// outstanding, a reject for the UE's request still matches when it arrives, and the SimUe reports
// a pass for whatever procedure is running by then -- the network's, which the reject answers
// nothing about.
func TestNetworkCommandAbortsTheUesOwnModification(t *testing.T) {
	ue, simUeChan := newUeWithOutstandingRequest(modificationRequestPTI)

	nasMsg := nas.NewMessage()
	nasMsg.GsmMessage = nas.NewGsmMessage()
	nasMsg.PDUSessionModificationCommand = nasMessage.NewPDUSessionModificationCommand(nas.MsgTypePDUSessionModificationCommand)
	nasMsg.PDUSessionModificationCommand.PDUSessionID.Octet = pduSessID
	nasMsg.PDUSessionModificationCommand.PTI.Octet = 0

	msg := &common.UeMessage{}
	msg.Event = common.PDU_SESS_MOD_COMMAND_EVENT
	msg.NasMsg = nasMsg

	if err := HandlePduSessModificationCompleteEvent(ue, msg); err != nil {
		t.Fatalf("answering the command returned: %v", err)
	}

	// Take the acknowledgement off the channel before the reject below: the SimUe channel holds
	// one message, and leaving it full turns the failure this test is looking for into a deadlock
	// rather than an assertion.
	select {
	case ack := <-simUeChan:
		if ack.GetEventType() != common.PDU_SESS_MOD_COMPLETE_EVENT {
			t.Fatalf("SimUe was told %v, want %v", ack.GetEventType(), common.PDU_SESS_MOD_COMPLETE_EVENT)
		}
	default:
		t.Fatal("the command was not acknowledged")
	}

	if got := ue.PduSessions[pduSessID].PendingPTI; got != 0 {
		t.Errorf("outstanding PTI = %d, want 0: the UE's own procedure is aborted by the collision", got)
	}

	// And with it forgotten, the reject that answers nothing can no longer be taken for an answer.
	if err := forwardDlNasToSimUe(ue, newRejectMessage(pduSessID, modificationRequestPTI),
		nas.MsgTypePDUSessionModificationReject); err == nil {
		t.Error("a reject for the aborted request was accepted; it would pass the network's procedure")
	}
}
