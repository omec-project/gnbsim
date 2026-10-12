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

func TestRanIDToNgap(t *testing.T) {
	ran := models.GlobalRanNodeId{PlmnId: models.PlmnId{Mcc: "208", Mnc: "93"}, GNbId: &models.GNbId{BitLength: 22, GNBValue: "000001"}}
	got := RanIDToNgap(ran)
	if got.Present != ngapType.GlobalRANNodeIDPresentGlobalGNBID || got.GlobalGNBID == nil {
		t.Fatalf("RanIDToNgap() = %#v", got)
	}
}
