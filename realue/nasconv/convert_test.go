// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nasconv

import (
	"testing"

	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/nasType"
	"github.com/omec-project/openapi/v2/models"
)

func TestGutiRoundTrip(t *testing.T) {
	want := "2089301020304050607"
	guti := GutiToNas(want)
	_, got := GutiToString(guti.Octet[:])
	if got != want {
		t.Fatalf("GUTI round trip = %q, want %q", got, want)
	}
}

func TestSpareHalfOctetAndNgksiToModels(t *testing.T) {
	ngKsi := nasType.SpareHalfOctetAndNgksi{}
	ngKsi.SetTSC(nasMessage.TypeOfSecurityContextFlagMapped)
	ngKsi.SetNasKeySetIdentifiler(3)
	got := SpareHalfOctetAndNgksiToModels(ngKsi)
	if got.Tsc != models.SCTYPE_MAPPED || got.Ksi != 3 {
		t.Fatalf("SpareHalfOctetAndNgksiToModels() = %#v", got)
	}
}

func TestPDUSessionTypeToModels(t *testing.T) {
	if got := PDUSessionTypeToModels(nasMessage.PDUSessionTypeIPv4IPv6); got != models.PDUSESSIONTYPE_IPV4_V6 {
		t.Fatalf("PDUSessionTypeToModels() = %q", got)
	}
}
