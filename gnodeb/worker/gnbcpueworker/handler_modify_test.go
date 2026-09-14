// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"testing"

	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/ngapType"
)

// releaseTransfer builds a modify request transfer carrying a QoS Flow to Release List, which is
// the shape the SMF sends when it withdraws a flow: that request names nothing else.
func releaseTransfer(qfis ...int64) *ngapType.PDUSessionResourceModifyRequestTransfer {
	list := &ngapType.QosFlowListWithCause{}
	for _, qfi := range qfis {
		// The cause is mandatory on the item, and the core's is why it withdrew the flow.
		item := ngapType.QosFlowWithCauseItem{
			QosFlowIdentifier: ngapType.QosFlowIdentifier{Value: qfi},
		}
		item.Cause.Present = ngapType.CausePresentRadioNetwork
		item.Cause.RadioNetwork = &ngapType.CauseRadioNetwork{
			Value: ngapType.CauseRadioNetworkPresentReleaseDueTo5gcGeneratedReason,
		}
		list.List = append(list.List, item)
	}

	transfer := &ngapType.PDUSessionResourceModifyRequestTransfer{}
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		ngapType.PDUSessionResourceModifyRequestTransferIEs{
			Id: ngapType.ProtocolIEID{Value: ngapType.ProtocolIEIDQosFlowToReleaseList},
			Value: ngapType.PDUSessionResourceModifyRequestTransferIEsValue{
				Present:              ngapType.PDUSessionResourceModifyRequestTransferIEsPresentQosFlowToReleaseList,
				QosFlowToReleaseList: list,
			},
		})
	return transfer
}

func TestReleasedQfis(t *testing.T) {
	tests := []struct {
		name     string
		transfer *ngapType.PDUSessionResourceModifyRequestTransfer
		want     []int64
	}{
		{
			name:     "a release-only request names the flows to drop",
			transfer: releaseTransfer(2, 3),
			want:     []int64{2, 3},
		},
		{
			name:     "a request with no release list releases nothing",
			transfer: &ngapType.PDUSessionResourceModifyRequestTransfer{},
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := releasedQfis(tc.transfer)
			if len(got) != len(tc.want) {
				t.Fatalf("releasedQfis() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("releasedQfis()[%d] = %d, want %d", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// addOrModifyTransfer builds a modify request transfer that asks for the named QoS flows, which is
// the ordinary shape: the SMF adds or changes flows and the gNB decides which it will admit.
func addOrModifyTransfer(qfis ...int64) *ngapType.PDUSessionResourceModifyRequestTransfer {
	list := &ngapType.QosFlowAddOrModifyRequestList{}
	for _, qfi := range qfis {
		list.List = append(list.List, ngapType.QosFlowAddOrModifyRequestItem{
			QosFlowIdentifier: ngapType.QosFlowIdentifier{Value: qfi},
		})
	}

	transfer := &ngapType.PDUSessionResourceModifyRequestTransfer{}
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		ngapType.PDUSessionResourceModifyRequestTransferIEs{
			Id: ngapType.ProtocolIEID{Value: ngapType.ProtocolIEIDQosFlowAddOrModifyRequestList},
			Value: ngapType.PDUSessionResourceModifyRequestTransferIEsValue{
				Present:                       ngapType.PDUSessionResourceModifyRequestTransferIEsPresentQosFlowAddOrModifyRequestList,
				QosFlowAddOrModifyRequestList: list,
			},
		})
	return transfer
}

// modifyItem wraps a transfer in the per-session item the request carries it in.
func modifyItem(t *testing.T,
	transfer *ngapType.PDUSessionResourceModifyRequestTransfer,
) *ngapType.PDUSessionResourceModifyItemModReq {
	t.Helper()
	encoded, err := aper.MarshalWithParams(*transfer, "valueExt")
	if err != nil {
		t.Fatalf("encoding the request transfer failed: %v", err)
	}
	return &ngapType.PDUSessionResourceModifyItemModReq{
		PDUSessionID:                            ngapType.PDUSessionID{Value: testPduSessID},
		PDUSessionResourceModifyRequestTransfer: encoded,
	}
}

const testPduSessID = 10

// newCpUe builds a gNB control plane context serving one PDU session with the given QoS flows.
func newCpUe(gnb *gnbctx.GNodeB, qfis ...int64) (*gnbctx.GnbCpUe, *gnbctx.GnbUpUe) {
	upCtx := &gnbctx.GnbUpUe{
		QosFlows:  make(map[int64]*ngapType.QosFlowSetupRequestItem),
		Log:       logger.GNodeBLog,
		PduSessId: testPduSessID,
	}
	for _, qfi := range qfis {
		upCtx.AddQosFlow(qfi, &ngapType.QosFlowSetupRequestItem{})
	}

	gnbue := &gnbctx.GnbCpUe{Gnb: gnb, Log: logger.GNodeBLog}
	gnbue.AddGnbUpUe(testPduSessID, upCtx)
	return gnbue, upCtx
}

// recordedQfis is what the gNB believes it is serving, which is what the uplink path stamps from.
func recordedQfis(upCtx *gnbctx.GnbUpUe, qfis ...int64) []bool {
	held := make([]bool, len(qfis))
	for i, qfi := range qfis {
		held[i] = upCtx.GetQosFlow(qfi) != nil
	}
	return held
}

// TestModifySessionAppliesWhatItReports pins the ordinary case: the answer and the gNB's own view
// of the session say the same thing about every flow the request named.
func TestModifySessionAppliesWhatItReports(t *testing.T) {
	gnb := &gnbctx.GNodeB{ModifyRejectQfis: []int64{2}}
	gnbue, upCtx := newCpUe(gnb, 1)

	encoded, cause := modifySession(gnbue, modifyItem(t, addOrModifyTransfer(2, 3)))
	if cause != nil {
		t.Fatalf("the session was failed with cause %v, want it modified", cause.Present)
	}
	if len(encoded) == 0 {
		t.Fatal("no response transfer was produced for a modified session")
	}

	if held := recordedQfis(upCtx, 1, 2, 3); held[0] != true || held[1] != false || held[2] != true {
		t.Errorf("recorded flows 1,2,3 = %v, want [true false true]: the refused flow must not be recorded",
			held)
	}
}

// TestModifySessionAppliesReleases covers the request the SMF builds when it withdraws a flow
// after a partial rejection: it names nothing but the flows to drop.
func TestModifySessionAppliesReleases(t *testing.T) {
	gnbue, upCtx := newCpUe(&gnbctx.GNodeB{}, 1, 2)

	_, cause := modifySession(gnbue, modifyItem(t, releaseTransfer(2)))
	if cause != nil {
		t.Fatalf("a release-only modification was failed with cause %v, want it modified", cause.Present)
	}

	if held := recordedQfis(upCtx, 1, 2); held[0] != true || held[1] != false {
		t.Errorf("recorded flows 1,2 = %v, want [true false]", held)
	}
}

// TestModifySessionLeavesTheSessionAloneWhenItFails is the invariant behind deciding before
// committing. A session reported as failed has its modification command withheld, so the core and
// the UE both go on holding it at its previous parameters — a gNB that had already applied the
// releases and admissions would be the only party that had moved, and nothing would tell it so.
func TestModifySessionLeavesTheSessionAloneWhenItFails(t *testing.T) {
	tests := []struct {
		gnb       *gnbctx.GNodeB
		item      func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq
		name      string
		wantCause aper.Enumerated
	}{
		{
			name: "the whole modification is refused",
			gnb:  &gnbctx.GNodeB{ModifyRejectAll: true},
			item: func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq {
				return modifyItem(t, releaseTransfer(2))
			},
			wantCause: ngapType.CauseRadioNetworkPresentRadioResourcesNotAvailable,
		},
		{
			name: "the request transfer will not decode",
			gnb:  &gnbctx.GNodeB{},
			item: func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq {
				return &ngapType.PDUSessionResourceModifyItemModReq{
					PDUSessionID: ngapType.PDUSessionID{Value: testPduSessID},
					// A container claiming an IE it does not carry: the decoder reports
					// "sequence truncated". Arbitrary bytes will not do -- aper decodes most of
					// them into a transfer with no IEs at all, which is a decode that succeeded.
					PDUSessionResourceModifyRequestTransfer: []byte{0x00, 0x00, 0x05},
				}
			},
			wantCause: ngapType.CauseRadioNetworkPresentRadioResourcesNotAvailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gnbue, upCtx := newCpUe(tc.gnb, 1, 2)

			encoded, cause := modifySession(gnbue, tc.item(t))
			if cause == nil {
				t.Fatal("the session was reported as modified, want failed")
			}
			if encoded != nil {
				t.Error("a failed session carries a response transfer")
			}
			if cause.RadioNetwork == nil || cause.RadioNetwork.Value != tc.wantCause {
				t.Errorf("cause = %v, want radio network %v", cause.RadioNetwork, tc.wantCause)
			}
			if held := recordedQfis(upCtx, 1, 2); held[0] != true || held[1] != true {
				t.Errorf("recorded flows 1,2 = %v, want both kept: a failed session changed nothing", held)
			}
		})
	}
}

// TestModifySessionFailsASessionItHoldsNoContextFor covers a request naming a session this gNB is
// not serving. Answering "admitted" would promise the core flows on a bearer that does not exist,
// and the UE would be told its QoS changed by a radio that never heard of the session.
//
// The answer is the same whatever else is wrong with the request, which is why the check comes
// first. Behind the configured refusal, an unknown session was reported as "radio resources not
// available" -- an answer that tells the core to retry later for a session that will never exist,
// and one a profile driving modifyRejectAll would produce for every mistyped session id.
func TestModifySessionFailsASessionItHoldsNoContextFor(t *testing.T) {
	tests := []struct {
		gnb  *gnbctx.GNodeB
		item func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq
		name string
	}{
		{
			name: "an ordinary request",
			gnb:  &gnbctx.GNodeB{},
			item: func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq {
				return modifyItem(t, addOrModifyTransfer(1))
			},
		},
		{
			name: "with the whole modification refused by configuration",
			gnb:  &gnbctx.GNodeB{ModifyRejectAll: true},
			item: func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq {
				return modifyItem(t, addOrModifyTransfer(1))
			},
		},
		{
			name: "with a transfer that will not decode",
			gnb:  &gnbctx.GNodeB{},
			item: func(t *testing.T) *ngapType.PDUSessionResourceModifyItemModReq {
				return &ngapType.PDUSessionResourceModifyItemModReq{
					PDUSessionID:                            ngapType.PDUSessionID{Value: testPduSessID},
					PDUSessionResourceModifyRequestTransfer: []byte{0x00, 0x00, 0x05},
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gnbue := &gnbctx.GnbCpUe{Gnb: tc.gnb, Log: logger.GNodeBLog}

			encoded, cause := modifySession(gnbue, tc.item(t))
			if cause == nil {
				t.Fatal("a session with no user plane context was reported as modified")
			}
			if encoded != nil {
				t.Error("a failed session carries a response transfer")
			}
			if cause.RadioNetwork == nil ||
				cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentUnknownPDUSessionID {
				t.Errorf("cause = %v, want radio network unknown-PDU-session-ID", cause.RadioNetwork)
			}
		})
	}
}

// TestDecideQosFlowsChangesNothing is the same invariant one level down, where it is enforced:
// deciding is separate from applying, so a decision can be discarded if the answer that reports it
// cannot be encoded.
func TestDecideQosFlowsChangesNothing(t *testing.T) {
	gnb := &gnbctx.GNodeB{ModifyRejectQfis: []int64{3}}
	gnbue, upCtx := newCpUe(gnb, 1, 2)

	transfer := addOrModifyTransfer(3, 4)
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		releaseTransfer(2).ProtocolIEs.List...)

	plan := decideQosFlows(gnbue, testPduSessID, transfer)
	if held := recordedQfis(upCtx, 1, 2, 4); held[0] != true || held[1] != true || held[2] != false {
		t.Fatalf("recorded flows 1,2,4 = %v after deciding, want [true true false]: deciding must not write",
			held)
	}

	plan.apply(upCtx)
	if held := recordedQfis(upCtx, 1, 2, 3, 4); held[0] != true || held[1] != false ||
		held[2] != false || held[3] != true {
		t.Errorf("recorded flows 1,2,3,4 = %v after applying, want [true false false true]", held)
	}
}
