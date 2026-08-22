// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"bytes"

	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/nasType"
)

// GetPduSessionModificationComplete builds a PDU SESSION MODIFICATION COMPLETE.
//
// TS 24.501 subclause 8.3.9. The PTI is echoed from the command rather than assumed: for a
// network-requested modification it is "no procedure transaction identity assigned" (0), and for
// the answer to a UE-requested one it is the identity the UE chose. Sending a fixed value would
// leave the network unable to match the acknowledgement to the procedure it started.
//
// github.com/omec-project/nas/v2 carries no builder for this message, only for the establishment
// and release ones, so it is built here.
func GetPduSessionModificationComplete(pduSessionId uint8, pti uint8) ([]byte, error) {
	m := nas.NewMessage()
	m.GsmMessage = nas.NewGsmMessage()
	m.GsmHeader.SetMessageType(nas.MsgTypePDUSessionModificationComplete)

	complete := nasMessage.NewPDUSessionModificationComplete(0)
	complete.SetExtendedProtocolDiscriminator(
		nasMessage.Epd5GSSessionManagementMessage)
	complete.SetMessageType(nas.MsgTypePDUSessionModificationComplete)
	complete.SetPDUSessionID(pduSessionId)
	complete.SetPTI(pti)

	m.PDUSessionModificationComplete = complete

	data := new(bytes.Buffer)
	if err := m.GsmMessageEncode(data); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

// GetUlNasTransportPduSessionModificationComplete wraps the complete in the UL NAS TRANSPORT that
// carries it to the AMF.
//
// The Request Type IE is deliberately absent. TS 24.501 subclause 8.2.10 makes it conditional, and
// it exists to tell the AMF what a UE wants done with a session it is establishing. Including it
// on an acknowledgement is what caused the AMF to treat a modification as a new establishment and
// tear the session down — the misclassification the AMF side of this work fixes.
func GetUlNasTransportPduSessionModificationComplete(pduSessionId uint8, pti uint8) ([]byte, error) {
	payload, err := GetPduSessionModificationComplete(pduSessionId, pti)
	if err != nil {
		return nil, err
	}

	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeULNASTransport)

	ulNasTransport := nasMessage.NewULNASTransport(0)
	ulNasTransport.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	ulNasTransport.SetMessageType(nas.MsgTypeULNASTransport)
	ulNasTransport.SetExtendedProtocolDiscriminator(
		nasMessage.Epd5GSMobilityManagementMessage)
	ulNasTransport.PduSessionID2Value = new(nasType.PduSessionID2Value)
	ulNasTransport.PduSessionID2Value.SetIei(nasMessage.ULNASTransportPduSessionID2ValueType)
	ulNasTransport.SetPduSessionID2Value(pduSessionId)

	ulNasTransport.SetPayloadContainerType(nasMessage.PayloadContainerTypeN1SMInfo)
	ulNasTransport.PayloadContainer.SetLen(uint16(len(payload)))
	ulNasTransport.SetPayloadContainerContents(payload)

	m.ULNASTransport = ulNasTransport

	data := new(bytes.Buffer)
	if err := m.GmmMessageEncode(data); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

// GetPduSessionModificationRequest builds a PDU SESSION MODIFICATION REQUEST.
//
// TS 24.501 subclause 8.3.7. Deliberately minimal: this exists so the network's answer can be
// verified, not so a modification can succeed. The core refuses every UE-requested modification,
// so there is nothing to be gained by proposing QoS rules it will never evaluate — and a request
// carrying none is still a well-formed request.
//
// The PTI is the UE's, and must be non-zero: zero means "no procedure transaction identity
// assigned" and marks a network-requested procedure, so a request sent with zero could not be
// matched to its answer.
func GetPduSessionModificationRequest(pduSessionId uint8, pti uint8) ([]byte, error) {
	m := nas.NewMessage()
	m.GsmMessage = nas.NewGsmMessage()
	m.GsmHeader.SetMessageType(nas.MsgTypePDUSessionModificationRequest)

	req := nasMessage.NewPDUSessionModificationRequest(0)
	req.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSSessionManagementMessage)
	req.SetMessageType(nas.MsgTypePDUSessionModificationRequest)
	req.SetPDUSessionID(pduSessionId)
	req.SetPTI(pti)

	m.PDUSessionModificationRequest = req

	data := new(bytes.Buffer)
	if err := m.GsmMessageEncode(data); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

// GetUlNasTransportPduSessionModificationRequest wraps the request in its UL NAS TRANSPORT.
//
// requestType is a parameter rather than a constant on purpose. TS 24.501 subclause 5.4.5.2.2
// carries the Request type IE on a modification request, and the correct value is
// "modification request". A UE that sets "initial request" instead makes the AMF read the message
// as an attempt to establish a session that already exists — which released the session before
// that was fixed. Being able to send either is what lets the AMF's handling of both be verified;
// passing zero omits the IE, which is a third thing a UE may do.
func GetUlNasTransportPduSessionModificationRequest(pduSessionId uint8, pti uint8,
	requestType uint8,
) ([]byte, error) {
	payload, err := GetPduSessionModificationRequest(pduSessionId, pti)
	if err != nil {
		return nil, err
	}

	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeULNASTransport)

	ulNasTransport := nasMessage.NewULNASTransport(0)
	ulNasTransport.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	ulNasTransport.SetMessageType(nas.MsgTypeULNASTransport)
	ulNasTransport.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)
	ulNasTransport.PduSessionID2Value = new(nasType.PduSessionID2Value)
	ulNasTransport.PduSessionID2Value.SetIei(nasMessage.ULNASTransportPduSessionID2ValueType)
	ulNasTransport.SetPduSessionID2Value(pduSessionId)

	if requestType != 0 {
		ulNasTransport.RequestType = new(nasType.RequestType)
		ulNasTransport.RequestType.SetIei(nasMessage.ULNASTransportRequestTypeType)
		ulNasTransport.SetRequestTypeValue(requestType)
	}

	ulNasTransport.SetPayloadContainerType(nasMessage.PayloadContainerTypeN1SMInfo)
	ulNasTransport.PayloadContainer.SetLen(uint16(len(payload)))
	ulNasTransport.SetPayloadContainerContents(payload)

	m.ULNASTransport = ulNasTransport

	data := new(bytes.Buffer)
	if err := m.GmmMessageEncode(data); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}
