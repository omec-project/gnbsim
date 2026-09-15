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

// The acknowledgement is what ends a network-requested modification, and the whole
// network-requested path depends on this builder: a malformed one leaves the core retransmitting
// and then abandoning, which reads as the core's defect rather than the UE's.
//
// Three things are checked because each has its own way of going wrong. The PTI is echoed from the
// command, so a builder that assumed a value could not answer a UE-requested modification. The
// payload has to be a PDU SESSION MODIFICATION COMPLETE for the session named, or the SMF has
// nothing to match. And the Request type IE has to be absent: it tells the AMF what a UE wants
// done with a session it is establishing, and putting it on an acknowledgement is the
// misclassification that released sessions.
func TestModificationCompleteCarriesTheCommandsPti(t *testing.T) {
	tests := []struct {
		name string
		pti  uint8
	}{
		{"network-requested, no procedure transaction identity assigned", 0},
		{"answering the UE's own request", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := GetUlNasTransportPduSessionModificationComplete(10, tc.pti)
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

			if ul.RequestType != nil {
				t.Errorf("Request type IE is present with value %d: an acknowledgement carries none",
					ul.GetRequestTypeValue())
			}
			if got := ul.GetPayloadContainerType(); got != nasMessage.PayloadContainerTypeN1SMInfo {
				t.Errorf("payload container type = %d, want N1 SM information", got)
			}
			if got := ul.GetPduSessionID2Value(); got != 10 {
				t.Errorf("PDU session ID 2 = %d, want 10", got)
			}

			payload := ul.GetPayloadContainerContents()
			inner := nas.NewMessage()
			if err := inner.GsmMessageDecode(&payload); err != nil {
				t.Fatalf("the payload container does not decode as a GSM message: %v", err)
			}
			if got := inner.GsmHeader.GetMessageType(); got != nas.MsgTypePDUSessionModificationComplete {
				t.Fatalf("payload message type = 0x%02x, want 0x%02x",
					got, nas.MsgTypePDUSessionModificationComplete)
			}
			complete := inner.PDUSessionModificationComplete
			if got := complete.PTI.Octet; got != tc.pti {
				t.Errorf("PTI = %d, want %d: the network matches the acknowledgement by it", got, tc.pti)
			}
			if got := complete.PDUSessionID.Octet; got != 10 {
				t.Errorf("PDU session id = %d, want 10", got)
			}
		})
	}
}

// The PTI is the only identity a UE-requested modification has, and zero is the value that means
// there is none. The doc comment has said so since this builder existed; nothing enforced it, so a
// caller passing zero got a well-formed message that the network would read as its own
// transaction, and the reject it earned could not be matched to the UE's procedure.
func TestModificationRequestRefusesAZeroPti(t *testing.T) {
	if _, err := GetPduSessionModificationRequest(10, 0); err == nil {
		t.Error("a request with PTI 0 was built; it cannot be matched to the answer it gets")
	}
	if _, err := GetUlNasTransportPduSessionModificationRequest(10, 0, 5); err == nil {
		t.Error("the UL NAS TRANSPORT wrapper built a request with PTI 0")
	}
	if _, err := GetPduSessionModificationRequest(10, 1); err != nil {
		t.Errorf("a request with a valid PTI was refused: %v", err)
	}
}
