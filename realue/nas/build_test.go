// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"testing"

	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/openapi/v2/models"
)

func TestGetUlNasTransportPduSessionEstablishmentRequest(t *testing.T) {
	encoded, err := GetUlNasTransportPduSessionEstablishmentRequest(
		10,
		nasMessage.ULNASTransportRequestTypeInitialRequest,
		"internet",
		&models.Snssai{Sst: 1, Sd: new("010203")},
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
	if got := transport.GetPduSessionID2Value(); got != 10 {
		t.Errorf("PDU session ID 2 = %d, want 10", got)
	}
	if transport.RequestType == nil {
		t.Error("UL NAS transport is missing request type")
	} else if transport.GetRequestTypeValue() != nasMessage.ULNASTransportRequestTypeInitialRequest {
		t.Errorf("request type = %d, want initial request", transport.GetRequestTypeValue())
	}
	if transport.DNN == nil {
		t.Error("UL NAS transport is missing DNN")
	} else if string(transport.GetDNN()) != "internet" {
		t.Errorf("DNN = %q, want %q", transport.GetDNN(), "internet")
	}
	if got := transport.GetPayloadContainerType(); got != nasMessage.PayloadContainerTypeN1SMInfo {
		t.Errorf("payload container type = %d, want N1 SM information", got)
	}

	payload := transport.GetPayloadContainerContents()
	inner := nas.NewMessage()
	if err := inner.GsmMessageDecode(&payload); err != nil {
		t.Fatalf("payload container does not decode as a GSM message: %v", err)
	}
	if got := inner.GsmHeader.GetMessageType(); got != nas.MsgTypePDUSessionEstablishmentRequest {
		t.Fatalf("payload message type = 0x%02x, want 0x%02x", got, nas.MsgTypePDUSessionEstablishmentRequest)
	}
	if got := inner.PDUSessionEstablishmentRequest.GetPDUSessionID(); got != 10 {
		t.Errorf("inner PDU session ID = %d, want 10", got)
	}
	if got := inner.PDUSessionEstablishmentRequest.GetPTI(); got != 1 {
		t.Errorf("inner PTI = %d, want 1", got)
	}
}

func TestGetUlNasTransportPduSessionEstablishmentRequestRejectsInvalidSNssai(t *testing.T) {
	tests := []struct {
		snssai *models.Snssai
		name   string
	}{
		{name: "malformed SD", snssai: &models.Snssai{Sst: 1, Sd: new("not-hex")}},
		{name: "short SD", snssai: &models.Snssai{Sst: 1, Sd: new("0102")}},
		{name: "long SD", snssai: &models.Snssai{Sst: 1, Sd: new("01020304")}},
		{name: "negative SST", snssai: &models.Snssai{Sst: -1}},
		{name: "oversized SST", snssai: &models.Snssai{Sst: 256}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := GetUlNasTransportPduSessionEstablishmentRequest(
				10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", tc.snssai,
			); err == nil {
				t.Error("builder accepted an invalid S-NSSAI")
			}
		})
	}
}

func TestGetUlNasTransportPduSessionEstablishmentRequestEncodesSSTOnlySNssai(t *testing.T) {
	encoded, err := GetUlNasTransportPduSessionEstablishmentRequest(
		10,
		nasMessage.ULNASTransportRequestTypeInitialRequest,
		"internet",
		&models.Snssai{Sst: 1},
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	m := nas.NewMessage()
	if err := m.PlainNasDecode(&encoded); err != nil {
		t.Fatalf("the message this UE produced does not decode: %v", err)
	}
	if m.ULNASTransport == nil || m.ULNASTransport.SNSSAI == nil {
		t.Fatal("UL NAS transport is missing S-NSSAI")
	}
	if got := m.ULNASTransport.SNSSAI.GetLen(); got != 1 {
		t.Errorf("S-NSSAI length = %d, want 1 for SST-only S-NSSAI", got)
	}
	if got := m.ULNASTransport.GetSST(); got != 1 {
		t.Errorf("SST = %d, want 1", got)
	}
}
