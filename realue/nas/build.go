// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"bytes"
	"encoding/hex"
	"fmt"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/gnbsim/realue/nasconv"
	"github.com/omec-project/gnbsim/util/nastestpacket"
	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/nasTestpacket"
	"github.com/omec-project/nas/v2/nasType"
	"github.com/omec-project/openapi/v2/models"
)

func GetServiceRequest(ue *realuectx.RealUe) ([]byte, error) {
	nasMsg := nastestpacket.BuildServiceRequest(nasMessage.ServiceTypeData)
	serviceRequest := nasMsg.ServiceRequest

	guti := nasconv.GutiToNas(ue.Guti)
	serviceRequest.SetTypeOfIdentity(nasMessage.MobileIdentity5GSType5gSTmsi)
	serviceRequest.SetAMFSetID(guti.GetAMFSetID())
	serviceRequest.SetAMFPointer(guti.GetAMFPointer())
	serviceRequest.SetTMSI5G(guti.GetTMSI5G())
	serviceRequest.SetNasKeySetIdentifiler(uint8(ue.NgKsi.Ksi))

	data := new(bytes.Buffer)
	err := nasMsg.GmmMessageEncode(data)
	if err != nil {
		return nil, fmt.Errorf("encode failed:+%v", err)
	}

	return data.Bytes(), nil
}

// GetUlNasTransportPduSessionEstablishmentRequest wraps a PDU session
// establishment request in an UL NAS transport message.
func GetUlNasTransportPduSessionEstablishmentRequest(pduSessionID uint8, requestType uint8,
	dnn string, snssai *models.Snssai,
) ([]byte, error) {
	payload := nasTestpacket.GetPduSessionEstablishmentRequest(pduSessionID)

	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeULNASTransport)

	transport := nasMessage.NewULNASTransport(0)
	transport.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	transport.SetMessageType(nas.MsgTypeULNASTransport)
	transport.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)
	transport.PduSessionID2Value = new(nasType.PduSessionID2Value)
	transport.PduSessionID2Value.SetIei(nasMessage.ULNASTransportPduSessionID2ValueType)
	transport.SetPduSessionID2Value(pduSessionID)
	transport.RequestType = new(nasType.RequestType)
	transport.RequestType.SetIei(nasMessage.ULNASTransportRequestTypeType)
	transport.SetRequestTypeValue(requestType)

	if dnn != "" {
		if len(dnn) > 254 {
			return nil, fmt.Errorf("DNN length %d exceeds maximum 254 bytes", len(dnn))
		}
		transport.DNN = new(nasType.DNN)
		transport.DNN.SetIei(nasMessage.ULNASTransportDNNType)
		transport.SetDNN([]byte(dnn))
	}
	if snssai != nil {
		if sst := snssai.GetSst(); sst < 0 || sst > 255 {
			return nil, fmt.Errorf("S-NSSAI SST %d is outside the one-octet range", sst)
		}
		var sd [3]uint8
		if encodedSD := snssai.GetSd(); encodedSD != "" {
			decoded, err := hex.DecodeString(encodedSD)
			if err != nil {
				return nil, fmt.Errorf("decode S-NSSAI SD: %w", err)
			}
			if len(decoded) != len(sd) {
				return nil, fmt.Errorf("invalid S-NSSAI SD length %d, want %d", len(decoded), len(sd))
			}
			copy(sd[:], decoded)
		}
		transport.SNSSAI = nasType.NewSNSSAI(nasMessage.ULNASTransportSNSSAIType)
		transport.SetSST(uint8(snssai.GetSst()))
		transport.SNSSAI.SetLen(4)
		transport.SetSD(sd)
	}

	transport.SetPayloadContainerType(nasMessage.PayloadContainerTypeN1SMInfo)
	transport.PayloadContainer.SetLen(uint16(len(payload)))
	transport.SetPayloadContainerContents(payload)
	m.ULNASTransport = transport

	data := new(bytes.Buffer)
	if err := m.GmmMessageEncode(data); err != nil {
		return nil, fmt.Errorf("encode UL NAS transport: %w", err)
	}
	return data.Bytes(), nil
}
