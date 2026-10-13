// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ngapconv

import (
	"testing"

	"github.com/omec-project/ngap/v2/ngapType"
	"github.com/omec-project/openapi/v2/models"
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
	got4, got6 := IPAddressToString(IPAddressToNgap("192.0.2.1", "2001:db8::1"))
	if got4 != "192.0.2.1" || got6 != "2001:db8::1" {
		t.Fatalf("IP round trip = (%q, %q)", got4, got6)
	}
}

func TestIPAddressToNgapRejectsMalformedAddresses(t *testing.T) {
	for _, testCase := range [][2]string{{"not-an-ip", ""}, {"", "not-an-ip"}, {"192.0.2.1", "not-an-ip"}} {
		got := IPAddressToNgap(testCase[0], testCase[1])
		if got.Value.BitLength != 0 || len(got.Value.Bytes) != 0 {
			t.Errorf("IPAddressToNgap(%q, %q) = %#v, want zero address", testCase[0], testCase[1], got.Value)
		}
	}
}

func TestRanIDToNgap(t *testing.T) {
	ran := models.GlobalRanNodeId{PlmnId: models.PlmnId{Mcc: "208", Mnc: "93"}, GNbId: &models.GNbId{BitLength: 22, GNBValue: "000001"}}
	got := RanIDToNgap(ran)
	if got.Present != ngapType.GlobalRANNodeIDPresentGlobalGNBID || got.GlobalGNBID == nil {
		t.Fatalf("RanIDToNgap() = %#v", got)
	}
}
