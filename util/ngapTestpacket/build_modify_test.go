// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package ngapTestpacket

import (
	"testing"

	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/ngapType"
)

func cause() ngapType.Cause {
	c := ngapType.Cause{}
	c.Present = ngapType.CausePresentRadioNetwork
	c.RadioNetwork = &ngapType.CauseRadioNetwork{
		Value: ngapType.CauseRadioNetworkPresentRadioResourcesNotAvailable,
	}
	return c
}

// A partial rejection has to name both halves. Reporting only what was admitted would leave the
// core unable to tell a refused flow from one the request never mentioned, and it is the
// difference between those two that decides whether the session needs realigning.
func TestModifyResponseTransferReportsAdmittedAndRefusedSeparately(t *testing.T) {
	encoded, err := BuildPDUSessionResourceModifyResponseTransfer([]QosFlowOutcome{
		{QfiValue: 1, Succeeded: true},
		{QfiValue: 2, Cause: cause()},
		{QfiValue: 3, Succeeded: true},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded ngapType.PDUSessionResourceModifyResponseTransfer
	if err := aper.UnmarshalWithParams(encoded, &decoded, "valueExt"); err != nil {
		t.Fatalf("the transfer this gNB produced does not decode: %v", err)
	}

	if decoded.QosFlowAddOrModifyResponseList == nil ||
		len(decoded.QosFlowAddOrModifyResponseList.List) != 2 {
		t.Fatalf("admitted flows = %v, want 2", decoded.QosFlowAddOrModifyResponseList)
	}
	if decoded.QosFlowFailedToAddOrModifyList == nil ||
		len(decoded.QosFlowFailedToAddOrModifyList.List) != 1 {
		t.Fatalf("refused flows = %v, want 1", decoded.QosFlowFailedToAddOrModifyList)
	}
	if got := decoded.QosFlowFailedToAddOrModifyList.List[0].QosFlowIdentifier.Value; got != 2 {
		t.Errorf("refused QFI = %d, want 2", got)
	}
}

// A modification that admits everything must not carry an empty failed list, and the reverse.
// An empty list is not the same as an absent one to a conformant peer.
func TestModifyResponseTransferOmitsTheListItDoesNotNeed(t *testing.T) {
	tests := []struct {
		name            string
		outcomes        []QosFlowOutcome
		wantAdmittedNil bool
		wantRefusedNil  bool
	}{
		{"all admitted", []QosFlowOutcome{{QfiValue: 1, Succeeded: true}}, false, true},
		{"all refused", []QosFlowOutcome{{QfiValue: 1, Cause: cause()}}, true, false},
		{"nothing named", nil, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := BuildPDUSessionResourceModifyResponseTransfer(tc.outcomes)
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}
			var decoded ngapType.PDUSessionResourceModifyResponseTransfer
			if err := aper.UnmarshalWithParams(encoded, &decoded, "valueExt"); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if (decoded.QosFlowAddOrModifyResponseList == nil) != tc.wantAdmittedNil {
				t.Errorf("admitted list present = %v, want absent = %v",
					decoded.QosFlowAddOrModifyResponseList != nil, tc.wantAdmittedNil)
			}
			if (decoded.QosFlowFailedToAddOrModifyList == nil) != tc.wantRefusedNil {
				t.Errorf("refused list present = %v, want absent = %v",
					decoded.QosFlowFailedToAddOrModifyList != nil, tc.wantRefusedNil)
			}
		})
	}
}

// The tunnel information stays absent. A modification that changes QoS does not move the tunnel,
// and this is the case that exposes whether the peer's codec treats the field as optional — the
// deviation in omec-project/ngap v2.1.3, where the field lacks its optional tag.
func TestModifyResponseTransferLeavesTunnelInformationAbsent(t *testing.T) {
	encoded, err := BuildPDUSessionResourceModifyResponseTransfer(
		[]QosFlowOutcome{{QfiValue: 5, Succeeded: true}})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded ngapType.PDUSessionResourceModifyResponseTransfer
	if err := aper.UnmarshalWithParams(encoded, &decoded, "valueExt"); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.DLNGUUPTNLInformation != nil || decoded.ULNGUUPTNLInformation != nil {
		t.Error("tunnel information was encoded for a QoS-only modification")
	}
}

// The response carries the identities the core matches on, and reports each session in the list
// that matches its fate.
func TestModifyResponseCarriesIdentitiesAndBothSessionLists(t *testing.T) {
	transfer, err := BuildPDUSessionResourceModifyResponseTransfer(
		[]QosFlowOutcome{{QfiValue: 1, Succeeded: true}})
	if err != nil {
		t.Fatalf("transfer encode failed: %v", err)
	}

	pdu := BuildPDUSessionResourceModifyResponse(42, 7,
		map[int64][]byte{10: transfer},
		map[int64]ngapType.Cause{11: cause()})

	if pdu.SuccessfulOutcome == nil {
		t.Fatal("the response is not a successful outcome")
	}
	if got := pdu.SuccessfulOutcome.ProcedureCode.Value; got != ngapType.ProcedureCodePDUSessionResourceModify {
		t.Errorf("procedure code = %d, want %d", got, ngapType.ProcedureCodePDUSessionResourceModify)
	}

	response := pdu.SuccessfulOutcome.Value.PDUSessionResourceModify
	if response == nil {
		t.Fatal("PDUSessionResourceModifyResponse is nil")
	}

	var sawAmfId, sawRanId, sawModified, sawFailed bool
	for _, ie := range response.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			sawAmfId = ie.Value.AMFUENGAPID.Value == 42
		case ngapType.ProtocolIEIDRANUENGAPID:
			sawRanId = ie.Value.RANUENGAPID.Value == 7
		case ngapType.ProtocolIEIDPDUSessionResourceModifyListModRes:
			sawModified = len(ie.Value.PDUSessionResourceModifyListModRes.List) == 1
		case ngapType.ProtocolIEIDPDUSessionResourceFailedToModifyListModRes:
			sawFailed = len(ie.Value.PDUSessionResourceFailedToModifyListModRes.List) == 1
		}
	}
	if !sawAmfId || !sawRanId {
		t.Errorf("identities not carried: amf=%v ran=%v — the core cannot match the response without them",
			sawAmfId, sawRanId)
	}
	if !sawModified {
		t.Error("the modified session list is missing")
	}
	if !sawFailed {
		t.Error("the failed session list is missing")
	}
}
