// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ngapconv

import (
	"bytes"
	"testing"

	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/ngapType"
	"github.com/omec-project/openapi/v2/models"
)

const (
	invalidIP  = "not-an-ip"
	sampleIPv4 = "192.0.2.1"
)

func TestPlmnRoundTrip(t *testing.T) {
	want := models.PlmnId{Mcc: "208", Mnc: "93"}
	got, err := PlmnIdToModels(PlmnIdToNgap(want))
	if err != nil || got != want {
		t.Fatalf("PLMN round trip = %#v, %v; want %#v", got, err, want)
	}
}

func TestPlmnIdToNgapRejectsMalformedDigits(t *testing.T) {
	got := PlmnIdToNgap(models.PlmnId{Mcc: "208", Mnc: "9g"})
	if len(got.Value) != 0 {
		t.Fatalf("PlmnIdToNgap() returned partial identity %#v", got.Value)
	}
}

func TestPlmnIdToNgapRejectsInvalidLengths(t *testing.T) {
	testCases := []models.PlmnId{
		{Mcc: "20", Mnc: "93"},
		{Mcc: "208", Mnc: "9"},
		{Mcc: "208", Mnc: "9300"},
	}
	for _, testCase := range testCases {
		if got := PlmnIdToNgap(testCase); len(got.Value) != 0 {
			t.Errorf("PlmnIdToNgap(%#v) = %#v, want zero identity", testCase, got.Value)
		}
	}
}

func TestSNssaiRoundTrip(t *testing.T) {
	sd := "010203"
	want := models.Snssai{Sst: 1, Sd: &sd}
	got, err := SNssaiToModels(SNssaiToNgap(want))
	if err != nil || got.GetSst() != want.GetSst() || got.GetSd() != want.GetSd() {
		t.Fatalf("S-NSSAI round trip = %#v, %v", got, err)
	}
}

func TestIPAddressRoundTrip(t *testing.T) {
	got4, got6 := IPAddressToString(IPAddressToNgap(sampleIPv4, "2001:db8::1"))
	if got4 != sampleIPv4 || got6 != "2001:db8::1" {
		t.Fatalf("IP round trip = (%q, %q)", got4, got6)
	}
}

func TestIPAddressToNgapRejectsMalformedAddresses(t *testing.T) {
	for _, testCase := range [][2]string{{invalidIP, ""}, {"", invalidIP}, {sampleIPv4, invalidIP}, {"", sampleIPv4}} {
		got := IPAddressToNgap(testCase[0], testCase[1])
		if got.Value.BitLength != 0 || len(got.Value.Bytes) != 0 {
			t.Errorf("IPAddressToNgap(%q, %q) = %#v, want zero address", testCase[0], testCase[1], got.Value)
		}
	}
}

func TestRanIDToNgapGNBID(t *testing.T) {
	plmn := models.PlmnId{Mcc: "208", Mnc: "93"}
	ran := models.GlobalRanNodeId{PlmnId: plmn, GNbId: models.NewGNbId(22, "0000a7")}
	got := RanIDToNgap(ran)
	if got.Present != ngapType.GlobalRANNodeIDPresentGlobalGNBID || got.GlobalGNBID == nil {
		t.Fatalf("RanIDToNgap() = %#v", got)
	}
	gnbID := got.GlobalGNBID.GNBID
	want := []byte{0x00, 0x00, 0xa4}
	if gnbID.Present != ngapType.GNBIDPresentGNBID || gnbID.GNBID == nil ||
		gnbID.GNBID.BitLength != 22 || !bytes.Equal(gnbID.GNBID.Bytes, want) {
		t.Fatalf("GNBID = %#v, want bytes %#v with bit length 22", gnbID, want)
	}
}

func TestRanIDToNgapNgENBVariants(t *testing.T) {
	plmn := models.PlmnId{Mcc: "208", Mnc: "93"}
	testCases := []struct {
		name      string
		ngeNbId   string
		wantBytes []byte
		present   int
		wantBits  uint64
	}{
		{
			name: "MacroNGeNB", ngeNbId: "MacroNGeNB-1234f",
			wantBytes: []byte{0x12, 0x34, 0xc0}, present: ngapType.NgENBIDPresentMacroNgENBID, wantBits: 18,
		},
		{
			name: "SMacroNGeNB", ngeNbId: "SMacroNGeNB-6789a",
			wantBytes: []byte{0x67, 0x89, 0xa0}, present: ngapType.NgENBIDPresentShortMacroNgENBID, wantBits: 20,
		},
		{
			name: "LMacroNGeNB", ngeNbId: "LMacroNGeNB-abcdef",
			wantBytes: []byte{0xab, 0xcd, 0xe8}, present: ngapType.NgENBIDPresentLongMacroNgENBID, wantBits: 21,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ran := models.GlobalRanNodeId{PlmnId: plmn}
			ran.SetNgeNbId(testCase.ngeNbId)
			got := RanIDToNgap(ran)
			if got.Present != ngapType.GlobalRANNodeIDPresentGlobalNgENBID || got.GlobalNgENBID == nil {
				t.Fatalf("RanIDToNgap(%q) = %#v", testCase.ngeNbId, got)
			}
			ngENBID := got.GlobalNgENBID.NgENBID
			var bits *aper.BitString
			switch testCase.present {
			case ngapType.NgENBIDPresentMacroNgENBID:
				bits = ngENBID.MacroNgENBID
			case ngapType.NgENBIDPresentShortMacroNgENBID:
				bits = ngENBID.ShortMacroNgENBID
			case ngapType.NgENBIDPresentLongMacroNgENBID:
				bits = ngENBID.LongMacroNgENBID
			}
			if ngENBID.Present != testCase.present || bits == nil ||
				bits.BitLength != testCase.wantBits || !bytes.Equal(bits.Bytes, testCase.wantBytes) {
				t.Fatalf("NgENBID(%q) = %#v, want present %d bytes %#v bits %d",
					testCase.ngeNbId, ngENBID, testCase.present, testCase.wantBytes, testCase.wantBits)
			}
		})
	}
}

func TestRanIDToNgapN3IWF(t *testing.T) {
	ran := models.GlobalRanNodeId{PlmnId: models.PlmnId{Mcc: "208", Mnc: "93"}}
	ran.SetN3IwfId("1a2b")
	got := RanIDToNgap(ran)
	if got.Present != ngapType.GlobalRANNodeIDPresentGlobalN3IWFID || got.GlobalN3IWFID == nil {
		t.Fatalf("RanIDToNgap() = %#v", got)
	}
	n3iwfID := got.GlobalN3IWFID.N3IWFID
	want := []byte{0x1a, 0x2b}
	if n3iwfID.Present != ngapType.N3IWFIDPresentN3IWFID || n3iwfID.N3IWFID == nil ||
		n3iwfID.N3IWFID.BitLength != 16 || !bytes.Equal(n3iwfID.N3IWFID.Bytes, want) {
		t.Fatalf("N3IWFID = %#v, want bytes %#v with bit length 16", n3iwfID, want)
	}
}

func TestRanIDToNgapRejectsMalformedPayloads(t *testing.T) {
	plmn := models.PlmnId{Mcc: "208", Mnc: "93"}
	testCases := []struct {
		ran  func() models.GlobalRanNodeId
		name string
	}{
		{name: "short GNBID payload", ran: func() models.GlobalRanNodeId {
			return models.GlobalRanNodeId{PlmnId: plmn, GNbId: models.NewGNbId(22, "0001")}
		}},
		{name: "malformed MacroNGeNB payload", ran: func() models.GlobalRanNodeId {
			ran := models.GlobalRanNodeId{PlmnId: plmn}
			ran.SetNgeNbId("MacroNGeNB-123")
			return ran
		}},
		{name: "malformed N3IWF payload", ran: func() models.GlobalRanNodeId {
			ran := models.GlobalRanNodeId{PlmnId: plmn}
			ran.SetN3IwfId("zz")
			return ran
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := RanIDToNgap(testCase.ran())
			if got.Present != ngapType.GlobalRANNodeIDPresentNothing {
				t.Errorf("RanIDToNgap() = %#v, want present Nothing", got)
			}
		})
	}
}
