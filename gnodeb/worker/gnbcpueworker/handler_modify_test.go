// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"testing"

	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
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

// withAmbr adds a PDU Session Aggregate Maximum Bit Rate IE to a transfer, which is how the SMF
// carries a session AMBR update alongside (or instead of) a QoS flow list.
func withAmbr(transfer *ngapType.PDUSessionResourceModifyRequestTransfer,
) *ngapType.PDUSessionResourceModifyRequestTransfer {
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		ngapType.PDUSessionResourceModifyRequestTransferIEs{
			Id: ngapType.ProtocolIEID{Value: ngapType.ProtocolIEIDPDUSessionAggregateMaximumBitRate},
			Value: ngapType.PDUSessionResourceModifyRequestTransferIEsValue{
				Present: ngapType.PDUSessionResourceModifyRequestTransferIEsPresentPDUSessionAggregateMaximumBitRate,
				PDUSessionAggregateMaximumBitRate: &ngapType.PDUSessionAggregateMaximumBitRate{
					PDUSessionAggregateMaximumBitRateDL: ngapType.BitRate{Value: 100000},
					PDUSessionAggregateMaximumBitRateUL: ngapType.BitRate{Value: 100000},
				},
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

// TestModifySessionFailsWhenNoRequestSucceeds covers TS 38.413 clause 8.2.3.2: a PDU session
// appears in the modified list only if at least one request in its transfer succeeded. A transfer
// whose only requests all failed -- whether refused by configuration or caught by the duplicate-
// QFI check -- is reported failed instead, and the session is left exactly as it was.
func TestModifySessionFailsWhenNoRequestSucceeds(t *testing.T) {
	tests := []struct {
		gnb       *gnbctx.GNodeB
		transfer  *ngapType.PDUSessionResourceModifyRequestTransfer
		name      string
		wantCause aper.Enumerated
	}{
		{
			name:      "the one flow named is refused",
			gnb:       &gnbctx.GNodeB{ModifyRejectQfis: []int64{2}},
			transfer:  addOrModifyTransfer(2),
			wantCause: ngapType.CauseRadioNetworkPresentRadioResourcesNotAvailable,
		},
		{
			name:      "the only flow named is repeated in the add-or-modify list",
			gnb:       &gnbctx.GNodeB{},
			transfer:  addOrModifyTransfer(2, 2),
			wantCause: ngapType.CauseRadioNetworkPresentMultipleQosFlowIDInstances,
		},
		{
			name:      "the only flow named is repeated in the release list",
			gnb:       &gnbctx.GNodeB{},
			transfer:  releaseTransfer(2, 2),
			wantCause: ngapType.CauseRadioNetworkPresentMultipleQosFlowIDInstances,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gnbue, upCtx := newCpUe(tc.gnb, 2)

			encoded, cause := modifySession(gnbue, modifyItem(t, tc.transfer))
			if cause == nil {
				t.Fatal("the session was reported as modified, want failed")
			}
			if encoded != nil {
				t.Error("a failed session carries a response transfer")
			}
			if cause.RadioNetwork == nil || cause.RadioNetwork.Value != tc.wantCause {
				t.Errorf("cause = %v, want radio network %v", cause.RadioNetwork, tc.wantCause)
			}
			if upCtx.GetQosFlow(2) == nil {
				t.Error("flow 2 is gone from the session; a failed session must change nothing")
			}
		})
	}
}

// TestModifySessionFailsWithUnspecifiedWhenMultipleFlowsFailForDifferentReasons covers the
// fallback cause: TS 38.413 gives no single cause for a session where every flow failed but not
// for the same reason, so this gNB reports the whole session refused rather than pick one flow's
// cause to stand for all of them.
func TestModifySessionFailsWithUnspecifiedWhenMultipleFlowsFailForDifferentReasons(t *testing.T) {
	gnbue, _ := newCpUe(&gnbctx.GNodeB{ModifyRejectQfis: []int64{2}}, 2, 3)

	transfer := addOrModifyTransfer(2, 3, 3)

	_, cause := modifySession(gnbue, modifyItem(t, transfer))
	if cause == nil {
		t.Fatal("the session was reported as modified, want failed")
	}
	if cause.RadioNetwork == nil ||
		cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentUnspecified {
		t.Errorf("cause = %v, want unspecified radio network failure", cause.RadioNetwork)
	}
}

// TestModifySessionSucceedsWhenAReleaseSucceedsDespiteEveryFlowFailing covers the other half of TS
// 38.413 clause 8.2.3.2: a release is itself a request the transfer names, so its success keeps the
// session modified even when every add-or-modify request in the same transfer failed.
func TestModifySessionSucceedsWhenAReleaseSucceedsDespiteEveryFlowFailing(t *testing.T) {
	gnbue, upCtx := newCpUe(&gnbctx.GNodeB{ModifyRejectQfis: []int64{2}}, 1)

	transfer := addOrModifyTransfer(2)
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		releaseTransfer(1).ProtocolIEs.List...)

	encoded, cause := modifySession(gnbue, modifyItem(t, transfer))
	if cause != nil {
		t.Fatalf("the session was failed with cause %v, want it modified: the release succeeded",
			cause.Present)
	}
	if len(encoded) == 0 {
		t.Fatal("no response transfer was produced for a modified session")
	}
	if upCtx.GetQosFlow(1) != nil {
		t.Error("flow 1 was not released despite the session being reported as modified")
	}
}

// TestModifySessionFailsWhenAReleaseListQfiIsRepeated covers TS 38.413 clause 8.2.3.2: NGAP has no
// QoS Flow Failed to Release List IE, so a repeated release-list QFI can only be made visible to
// the core by failing the session as a whole -- even though an add-or-modify request elsewhere in
// the same transfer succeeds, which would otherwise have kept the session modified.
func TestModifySessionFailsWhenAReleaseListQfiIsRepeated(t *testing.T) {
	gnbue, upCtx := newCpUe(&gnbctx.GNodeB{}, 5)

	transfer := addOrModifyTransfer(1)
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		releaseTransfer(5, 5).ProtocolIEs.List...)

	encoded, cause := modifySession(gnbue, modifyItem(t, transfer))
	if cause == nil {
		t.Fatal("the session was reported as modified, want failed: QFI 5's repeated release has no other way to be reported")
	}
	if encoded != nil {
		t.Error("a failed session carries a response transfer")
	}
	if cause.RadioNetwork == nil ||
		cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentMultipleQosFlowIDInstances {
		t.Errorf("cause = %v, want multiple-QoS-flow-ID-instances", cause.RadioNetwork)
	}
	if upCtx.GetQosFlow(1) != nil {
		t.Error("QFI 1 was admitted despite the session being failed")
	}
	if upCtx.GetQosFlow(5) == nil {
		t.Error("QFI 5 was released despite the session being failed")
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

// tunnelModifyTransfer builds a request that moves the session's uplink tunnel, alongside a QoS
// change the gNB would otherwise admit.
func tunnelModifyTransfer(qfi int64) *ngapType.PDUSessionResourceModifyRequestTransfer {
	transfer := addOrModifyTransfer(qfi)

	gtpTunnel := &ngapType.GTPTunnel{}
	gtpTunnel.TransportLayerAddress.Value = aper.BitString{Bytes: []byte{10, 0, 0, 1}, BitLength: 32}
	gtpTunnel.GTPTEID.Value = aper.OctetString{0x00, 0x00, 0x00, 0x02}

	item := ngapType.ULNGUUPTNLModifyItem{}
	item.ULNGUUPTNLInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel
	item.ULNGUUPTNLInformation.GTPTunnel = gtpTunnel
	item.DLNGUUPTNLInformation.Present = ngapType.UPTransportLayerInformationPresentGTPTunnel
	item.DLNGUUPTNLInformation.GTPTunnel = gtpTunnel

	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		ngapType.PDUSessionResourceModifyRequestTransferIEs{
			Id: ngapType.ProtocolIEID{Value: ngapType.ProtocolIEIDULNGUUPTNLModifyList},
			Value: ngapType.PDUSessionResourceModifyRequestTransferIEsValue{
				Present:              ngapType.PDUSessionResourceModifyRequestTransferIEsPresentULNGUUPTNLModifyList,
				ULNGUUPTNLModifyList: &ngapType.ULNGUUPTNLModifyList{List: []ngapType.ULNGUUPTNLModifyItem{item}},
			},
		})
	return transfer
}

// TestModifySessionRefusesAMoveOfTheUplinkTunnel covers the request this gNB cannot carry out. The
// UL NG-U UP TNL Modify List names the tunnel it is to send uplink to from now on; this gNB keeps
// the tunnel it has, so answering "modified" would leave the core believing the uplink had moved
// while the packets went on arriving nowhere -- with no report to say otherwise.
func TestModifySessionRefusesAMoveOfTheUplinkTunnel(t *testing.T) {
	gnbue, upCtx := newCpUe(&gnbctx.GNodeB{}, 1)

	encoded, cause := modifySession(gnbue, modifyItem(t, tunnelModifyTransfer(2)))
	if cause == nil {
		t.Fatal("a request moving the uplink tunnel was reported as modified")
	}
	if encoded != nil {
		t.Error("a failed session carries a response transfer")
	}
	if cause.RadioNetwork == nil ||
		cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentUnspecified {
		t.Errorf("cause = %v, want radio network unspecified", cause.RadioNetwork)
	}
	if held := recordedQfis(upCtx, 1, 2); held[0] != true || held[1] != false {
		t.Errorf("recorded flows 1,2 = %v, want [true false]: a refused session changed nothing", held)
	}
}

// A request may name a flow without restating its QoS parameters, which is permitted and means the
// ones it already has. Recording a zero-value struct for that case emptied the gNB's own view of a
// flow it is still serving -- characteristics and ARP gone, on a modification that said nothing
// about them.
func TestModifySessionKeepsQosParametersAModificationOmits(t *testing.T) {
	gnbue, upCtx := newCpUe(&gnbctx.GNodeB{})

	established := &ngapType.QosFlowSetupRequestItem{
		QosFlowIdentifier: ngapType.QosFlowIdentifier{Value: 1},
	}
	established.QosFlowLevelQosParameters.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentNonDynamic5QI
	established.QosFlowLevelQosParameters.QosCharacteristics.NonDynamic5QI = &ngapType.NonDynamic5QIDescriptor{
		FiveQI: ngapType.FiveQI{Value: 9},
	}
	established.QosFlowLevelQosParameters.AllocationAndRetentionPriority.PriorityLevelARP.Value = 7
	upCtx.AddQosFlow(1, established)

	if _, cause := modifySession(gnbue, modifyItem(t, addOrModifyTransfer(1))); cause != nil {
		t.Fatalf("the session was failed with cause %v", cause.Present)
	}

	got := upCtx.GetQosFlow(1)
	if got == nil {
		t.Fatal("the flow is gone from the session")
	}
	if got.QosFlowLevelQosParameters.QosCharacteristics.NonDynamic5QI == nil {
		t.Fatal("the 5QI the flow was set up with was erased by a modification that did not name it")
	}
	if fiveQi := got.QosFlowLevelQosParameters.QosCharacteristics.NonDynamic5QI.FiveQI.Value; fiveQi != 9 {
		t.Errorf("5QI = %d, want 9", fiveQi)
	}
	if arp := got.QosFlowLevelQosParameters.AllocationAndRetentionPriority.PriorityLevelARP.Value; arp != 7 {
		t.Errorf("ARP priority = %d, want 7", arp)
	}
}

// TestDuplicatePduSessionIds covers TS 38.413 clause 8.2.3.4: a PDU Session ID named more than
// once has every occurrence failed, rather than the loop silently keeping whichever it saw last.
func TestDuplicatePduSessionIds(t *testing.T) {
	items := []ngapType.PDUSessionResourceModifyItemModReq{
		{PDUSessionID: ngapType.PDUSessionID{Value: 10}},
		{PDUSessionID: ngapType.PDUSessionID{Value: 11}},
		{PDUSessionID: ngapType.PDUSessionID{Value: 10}},
	}

	got := duplicatePduSessionIds(items)
	if !got[10] {
		t.Error("PDU session 10 appears twice and must be reported as duplicated")
	}
	if got[11] {
		t.Error("PDU session 11 appears once and must not be reported as duplicated")
	}
}

// addOrModifyTransferWithItems is like addOrModifyTransfer but lets a test attach QoS parameters
// to an item, which the GBR and delay-critical checks decide on.
func addOrModifyTransferWithItems(items ...ngapType.QosFlowAddOrModifyRequestItem,
) *ngapType.PDUSessionResourceModifyRequestTransfer {
	transfer := &ngapType.PDUSessionResourceModifyRequestTransfer{}
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		ngapType.PDUSessionResourceModifyRequestTransferIEs{
			Id: ngapType.ProtocolIEID{Value: ngapType.ProtocolIEIDQosFlowAddOrModifyRequestList},
			Value: ngapType.PDUSessionResourceModifyRequestTransferIEsValue{
				Present:                       ngapType.PDUSessionResourceModifyRequestTransferIEsPresentQosFlowAddOrModifyRequestList,
				QosFlowAddOrModifyRequestList: &ngapType.QosFlowAddOrModifyRequestList{List: items},
			},
		})
	return transfer
}

// TestDecideQosFlowsFailsOverlappingQfiWithoutReleasingIt covers TS 38.413 clause 8.2.3.4: a QFI
// named in both the add-or-modify and release lists is failed rather than admitted, and is left on
// the session rather than released.
func TestDecideQosFlowsFailsOverlappingQfiWithoutReleasingIt(t *testing.T) {
	gnbue, upCtx := newCpUe(&gnbctx.GNodeB{}, 5)

	transfer := addOrModifyTransfer(5, 6)
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		releaseTransfer(5).ProtocolIEs.List...)

	plan := decideQosFlows(gnbue, testPduSessID, transfer)

	var failedFive, succeededSix bool
	for _, o := range plan.outcomes {
		switch o.QfiValue {
		case 5:
			if o.Succeeded {
				t.Error("QFI 5 is named in both lists and must be reported as failed, not succeeded")
			}
			if o.Cause.RadioNetwork == nil ||
				o.Cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentMultipleQosFlowIDInstances {
				t.Errorf("QFI 5 cause = %v, want multiple-QoS-flow-ID-instances", o.Cause.RadioNetwork)
			}
			failedFive = true
		case 6:
			succeededSix = o.Succeeded
		}
	}
	if !failedFive {
		t.Fatal("QFI 5 outcome is missing")
	}
	if !succeededSix {
		t.Error("QFI 6, named only in the add-or-modify list, should still succeed")
	}
	for _, qfi := range plan.released {
		if qfi == 5 {
			t.Error("QFI 5 must not be released: it already exists and the conflict leaves it in place")
		}
	}

	plan.apply(upCtx)
	if upCtx.GetQosFlow(5) == nil {
		t.Error("QFI 5 was released despite the conflict; it should have been left exactly as it was")
	}
}

// TestDecideQosFlowsFailsRepeatedAddQfi covers TS 38.413 clause 8.2.3.4: a QFI named more than
// once in the add-or-modify list is failed once, rather than admitted twice or reported twice.
func TestDecideQosFlowsFailsRepeatedAddQfi(t *testing.T) {
	gnbue, _ := newCpUe(&gnbctx.GNodeB{})

	transfer := addOrModifyTransfer(5, 5, 6)
	plan := decideQosFlows(gnbue, testPduSessID, transfer)

	var failedFiveCount int
	var succeededSix bool
	for _, o := range plan.outcomes {
		switch o.QfiValue {
		case 5:
			failedFiveCount++
			if o.Succeeded {
				t.Error("QFI 5 is named twice and must be reported as failed, not succeeded")
			}
			if o.Cause.RadioNetwork == nil ||
				o.Cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentMultipleQosFlowIDInstances {
				t.Errorf("QFI 5 cause = %v, want multiple-QoS-flow-ID-instances", o.Cause.RadioNetwork)
			}
		case 6:
			succeededSix = o.Succeeded
		}
	}
	if failedFiveCount != 1 {
		t.Fatalf("QFI 5 outcome count = %d, want exactly 1", failedFiveCount)
	}
	if !succeededSix {
		t.Error("QFI 6, named once, should still succeed")
	}
	for _, flow := range plan.admitted {
		if flow.qfi == 5 {
			t.Error("QFI 5 must not be admitted: it is repeated in the add-or-modify list")
		}
	}
}

// TestDecideQosFlowsFailsRepeatedReleaseQfi covers TS 38.413 clause 8.2.3.4: a QFI named more than
// once in the release list is failed once, rather than silently released twice. It is counted in
// failedReleases rather than outcomes: NGAP has no QoS Flow Failed to Release List IE to encode it
// into, and it was never named in the add-or-modify list either.
func TestDecideQosFlowsFailsRepeatedReleaseQfi(t *testing.T) {
	gnbue, _ := newCpUe(&gnbctx.GNodeB{}, 5)

	transfer := releaseTransfer(5, 5)
	plan := decideQosFlows(gnbue, testPduSessID, transfer)

	if len(plan.outcomes) != 0 {
		t.Errorf("outcomes = %+v, want none: a release-list failure has no add-or-modify IE to be encoded into",
			plan.outcomes)
	}
	if len(plan.failedReleases) != 1 || plan.failedReleases[0] != 5 {
		t.Errorf("failedReleases = %v, want [5]", plan.failedReleases)
	}
	for _, qfi := range plan.released {
		if qfi == 5 {
			t.Error("QFI 5 must not be released: it is repeated in the release list")
		}
	}
}

// TestInvalidQosParameters covers the three abnormal conditions TS 38.413 clause 8.2.3.4 requires
// failing a QoS flow for: a GBR 5QI with no GBR QoS Flow Information, a non-GBR 5QI with one, and
// delay critical with no Maximum Data Burst Volume.
func TestInvalidQosParameters(t *testing.T) {
	if _, invalid := invalidQosParameters(nil); invalid {
		t.Error("a nil IE carries no parameters to check and must not be reported invalid")
	}

	gbrParams := &ngapType.QosFlowLevelQosParameters{}
	gbrParams.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentNonDynamic5QI
	gbrParams.QosCharacteristics.NonDynamic5QI = &ngapType.NonDynamic5QIDescriptor{
		FiveQI: ngapType.FiveQI{Value: 1}, // conversational voice: GBR
	}
	if _, invalid := invalidQosParameters(gbrParams); !invalid {
		t.Error("a GBR 5QI without GBR QoS Flow Information must be reported invalid")
	}
	gbrParams.GBRQosInformation = &ngapType.GBRQosInformation{}
	if _, invalid := invalidQosParameters(gbrParams); invalid {
		t.Error("the same flow with GBR QoS Flow Information attached must not be reported invalid")
	}

	nonGbrParams := &ngapType.QosFlowLevelQosParameters{}
	nonGbrParams.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentNonDynamic5QI
	nonGbrParams.QosCharacteristics.NonDynamic5QI = &ngapType.NonDynamic5QIDescriptor{
		FiveQI: ngapType.FiveQI{Value: 9}, // non-GBR
	}
	if _, invalid := invalidQosParameters(nonGbrParams); invalid {
		t.Error("a non-GBR 5QI needs no GBR QoS Flow Information")
	}
	nonGbrParams.GBRQosInformation = &ngapType.GBRQosInformation{}
	if _, invalid := invalidQosParameters(nonGbrParams); !invalid {
		t.Error("a non-GBR 5QI with GBR QoS Flow Information attached must be reported invalid")
	}

	// A dynamically assigned 5QI is used for the non-GBR resource type too (TS 23.501 subclause
	// 5.7.1.3), so a dynamic descriptor that is not marked delay critical must not be treated as
	// GBR on the choice discriminator alone.
	nonGbrDynamicParams := &ngapType.QosFlowLevelQosParameters{}
	nonGbrDynamicParams.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentDynamic5QI
	nonGbrDynamicParams.QosCharacteristics.Dynamic5QI = &ngapType.Dynamic5QIDescriptor{}
	if _, invalid := invalidQosParameters(nonGbrDynamicParams); invalid {
		t.Error("a dynamically assigned 5QI with no delay-critical marking needs no GBR QoS Flow Information")
	}
	// The converse check is standardized-5QI-only for the same reason: a dynamic descriptor's own
	// GBR QoS Flow Information cannot be called wrong when nothing here says the flow is non-GBR.
	nonGbrDynamicParams.GBRQosInformation = &ngapType.GBRQosInformation{}
	if _, invalid := invalidQosParameters(nonGbrDynamicParams); invalid {
		t.Error("a dynamically assigned 5QI with GBR QoS Flow Information attached must not be reported invalid")
	}

	delayCritical := &ngapType.QosFlowLevelQosParameters{}
	delayCritical.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentDynamic5QI
	delayCritical.QosCharacteristics.Dynamic5QI = &ngapType.Dynamic5QIDescriptor{
		DelayCritical: &ngapType.DelayCritical{Value: ngapType.DelayCriticalPresentDelayCritical},
	}
	delayCritical.GBRQosInformation = &ngapType.GBRQosInformation{} // satisfies the GBR check above
	if _, invalid := invalidQosParameters(delayCritical); !invalid {
		t.Error("delay critical without Maximum Data Burst Volume must be reported invalid")
	}
	delayCritical.QosCharacteristics.Dynamic5QI.MaximumDataBurstVolume = &ngapType.MaximumDataBurstVolume{}
	if _, invalid := invalidQosParameters(delayCritical); invalid {
		t.Error("the same flow with Maximum Data Burst Volume attached must not be reported invalid")
	}
}

// TestDecideQosFlowsFailsGbrFlowMissingGbrQosInformation is TestInvalidQosParameters' first case
// wired through decideQosFlows, pinning that the outcome carries the cause and admits nothing.
func TestDecideQosFlowsFailsGbrFlowMissingGbrQosInformation(t *testing.T) {
	gnbue, _ := newCpUe(&gnbctx.GNodeB{})

	params := &ngapType.QosFlowLevelQosParameters{}
	params.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentNonDynamic5QI
	params.QosCharacteristics.NonDynamic5QI = &ngapType.NonDynamic5QIDescriptor{
		FiveQI: ngapType.FiveQI{Value: 1},
	}
	params.AllocationAndRetentionPriority.PriorityLevelARP.Value = 1
	transfer := addOrModifyTransferWithItems(ngapType.QosFlowAddOrModifyRequestItem{
		QosFlowIdentifier:         ngapType.QosFlowIdentifier{Value: 7},
		QosFlowLevelQosParameters: params,
	})

	plan := decideQosFlows(gnbue, testPduSessID, transfer)
	if len(plan.admitted) != 0 {
		t.Fatal("a GBR flow with no GBR QoS Flow Information must not be admitted")
	}
	if len(plan.outcomes) != 1 || plan.outcomes[0].Succeeded {
		t.Fatalf("outcomes = %+v, want one failed outcome", plan.outcomes)
	}
	if plan.outcomes[0].Cause.RadioNetwork == nil ||
		plan.outcomes[0].Cause.RadioNetwork.Value != ngapType.CauseRadioNetworkPresentInvalidQosCombination {
		t.Errorf("cause = %v, want invalid-QoS-combination", plan.outcomes[0].Cause.RadioNetwork)
	}
}

// TestInvalidQosParametersNonDynamicDelayCritical covers the standardized-5QI counterpart of
// TestInvalidQosParameters' delay-critical case: a NonDynamic5QI descriptor using one of the
// delay-critical GBR values (TS 23.501 table 5.7.4-1, 82-90) also needs Maximum Data Burst Volume,
// not only a Dynamic5QI descriptor marked delay critical.
func TestInvalidQosParametersNonDynamicDelayCritical(t *testing.T) {
	params := &ngapType.QosFlowLevelQosParameters{}
	params.QosCharacteristics.Present = ngapType.QosCharacteristicsPresentNonDynamic5QI
	params.QosCharacteristics.NonDynamic5QI = &ngapType.NonDynamic5QIDescriptor{
		FiveQI: ngapType.FiveQI{Value: 82}, // delay-critical GBR
	}
	params.GBRQosInformation = &ngapType.GBRQosInformation{} // satisfies the GBR check

	if _, invalid := invalidQosParameters(params); !invalid {
		t.Error("a standardized delay-critical 5QI without Maximum Data Burst Volume must be reported invalid")
	}
	params.QosCharacteristics.NonDynamic5QI.MaximumDataBurstVolume = &ngapType.MaximumDataBurstVolume{}
	if _, invalid := invalidQosParameters(params); invalid {
		t.Error("the same flow with Maximum Data Burst Volume attached must not be reported invalid")
	}
}

// TestDecideQosFlowsTracksSessionAmbr pins that decideQosFlows sees the PDU Session Aggregate
// Maximum Bit Rate IE regardless of what the QoS flow lists in the same transfer contain: it is
// what lets sessionFailureCause tell an AMBR-only success apart from a transfer with nothing that
// succeeded at all.
func TestDecideQosFlowsTracksSessionAmbr(t *testing.T) {
	gnbue, _ := newCpUe(&gnbctx.GNodeB{})

	if plan := decideQosFlows(gnbue, testPduSessID, addOrModifyTransfer(1)); plan.ambrRequested {
		t.Error("a transfer with no AMBR IE must not be tracked as requesting one")
	}
	if plan := decideQosFlows(gnbue, testPduSessID, withAmbr(addOrModifyTransfer(1))); !plan.ambrRequested {
		t.Error("a transfer carrying the AMBR IE must be tracked as requesting one")
	}
}

// TestSessionFailureCauseAmbrKeepsSessionModified covers TS 38.413 clause 8.2.3.2: a session AMBR
// request succeeds independently of the radio's decision about the flows named in the same
// transfer, so refusing every one of those flows must not fail the session when AMBR was also
// requested.
func TestSessionFailureCauseAmbrKeepsSessionModified(t *testing.T) {
	failedOutcome := qosFlowPlan{
		outcomes: []ngapTestpacket.QosFlowOutcome{{QfiValue: 1, Cause: radioResourcesNotAvailable()}},
	}
	if cause := sessionFailureCause(failedOutcome); cause == nil {
		t.Fatal("a transfer with only failed QoS flows and no AMBR request must fail the session")
	}

	failedOutcome.ambrRequested = true
	if cause := sessionFailureCause(failedOutcome); cause != nil {
		t.Errorf("cause = %v, want nil: the AMBR request succeeds even though every QoS flow was refused",
			cause.RadioNetwork)
	}
}

// TestModifySessionAdmitsAmbrOnlyRequestWithAllFlowsRefused wires
// TestSessionFailureCauseAmbrKeepsSessionModified through modifySession, pinning that the session
// is reported modified -- and its NAS command therefore not withheld -- when AMBR succeeds even
// though the configured refusal rejects every named QoS flow.
func TestModifySessionAdmitsAmbrOnlyRequestWithAllFlowsRefused(t *testing.T) {
	gnb := &gnbctx.GNodeB{ModifyRejectQfis: []int64{1}}
	gnbue, _ := newCpUe(gnb, 1)

	transfer := withAmbr(addOrModifyTransfer(1))
	encoded, cause := modifySession(gnbue, modifyItem(t, transfer))
	if cause != nil {
		t.Fatalf("the session was failed with cause %v, want it modified: AMBR succeeds independently of the refused flow",
			cause.RadioNetwork)
	}
	if len(encoded) == 0 {
		t.Error("a modified session must encode a response transfer")
	}
}
