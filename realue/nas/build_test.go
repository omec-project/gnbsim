// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"testing"

	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
)

func TestGetUlNasTransportPduSessionEstablishmentRequest(t *testing.T) {
	encoded, err := GetUlNasTransportPduSessionEstablishmentRequest(
		10,
		nasMessage.ULNASTransportRequestTypeInitialRequest,
		"internet",
		&models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")},
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	m := nas.NewMessage()
	if err := m.PlainNasDecode(&encoded); err != nil {
		t.Fatalf("the message this UE produced does not decode: %v", err)
	}
	transport := m.ULNASTransport
	if transport == nil || transport.SNSSAI == nil {
		t.Fatal("UL NAS transport is missing S-NSSAI")
	}
	if got := transport.GetSST(); got != 1 {
		t.Errorf("SST = %d, want 1", got)
	}
	if got := transport.GetSD(); got != [3]uint8{1, 2, 3} {
		t.Errorf("SD = %#v, want [3]uint8{1, 2, 3}", got)
	}
}
