// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"testing"

	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
)

// The Request type IE is what decides whether the AMF reads a modification request as a
// modification or as an attempt to establish a session that already exists. Sending the wrong
// value released the session before that was fixed, so a simulator has to be able to send each
// value on purpose — and this pins that it actually does.
func TestModificationRequestCarriesTheConfiguredRequestType(t *testing.T) {
	tests := []struct {
		name        string
		requestType uint8
		wantPresent bool
		wantValue   uint8
	}{
		{"modification request, the correct value", nasMessage.ULNASTransportRequestTypeModificationRequest, true, 5},
		{"initial request, the misclassification hazard", nasMessage.ULNASTransportRequestTypeInitialRequest, true, 1},
		{"omitted", 0, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := GetUlNasTransportPduSessionModificationRequest(10, 1, tc.requestType)
			if err != nil {
				t.Fatalf("build failed: %v", err)
			}

			m := nas.NewMessage()
			if err := m.PlainNasDecode(&encoded); err != nil {
				t.Fatalf("the message this UE produced does not decode: %v", err)
			}
			ul := m.ULNASTransport
			if ul == nil {
				t.Fatal("not a UL NAS TRANSPORT")
			}

			if (ul.RequestType != nil) != tc.wantPresent {
				t.Fatalf("Request type IE present = %v, want %v", ul.RequestType != nil, tc.wantPresent)
			}
			if tc.wantPresent && ul.GetRequestTypeValue() != tc.wantValue {
				t.Errorf("Request type = %d, want %d", ul.GetRequestTypeValue(), tc.wantValue)
			}
		})
	}
}

// The payload has to be a modification request at the UE's own PTI. A zero PTI means "no procedure
// transaction identity assigned", which marks a network-requested procedure — a request sent that
// way could not be matched to the answer it gets.
func TestModificationRequestCarriesANonZeroPti(t *testing.T) {
	encoded, err := GetPduSessionModificationRequest(10, 7)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	m := nas.NewMessage()
	if err := m.GsmMessageDecode(&encoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got := m.GsmHeader.GetMessageType(); got != nas.MsgTypePDUSessionModificationRequest {
		t.Fatalf("message type = 0x%02x, want 0x%02x", got, nas.MsgTypePDUSessionModificationRequest)
	}
	req := m.PDUSessionModificationRequest
	if got := req.PTI.Octet; got != 7 {
		t.Errorf("PTI = %d, want 7", got)
	}
	if got := req.PDUSessionID.Octet; got != 10 {
		t.Errorf("PDU session id = %d, want 10", got)
	}
}
