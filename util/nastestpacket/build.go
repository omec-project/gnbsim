// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package nastestpacket

import (
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasType"
)

func BuildServiceRequest(serviceType uint8) *nas.Message {
	m := nas.NewMessage()
	m.GmmMessage = nas.NewGmmMessage()
	m.GmmHeader.SetMessageType(nas.MsgTypeServiceRequest)

	serviceRequest := nasMessage.NewServiceRequest(0)
	serviceRequest.SetExtendedProtocolDiscriminator(nasMessage.Epd5GSMobilityManagementMessage)
	serviceRequest.SetSecurityHeaderType(nas.SecurityHeaderTypePlainNas)
	serviceRequest.SetMessageType(nas.MsgTypeServiceRequest)
	serviceRequest.SetServiceTypeValue(serviceType)
	serviceRequest.SetNasKeySetIdentifiler(0x01)
	serviceRequest.SetAMFSetID(uint16(0xFE) << 2)
	serviceRequest.SetAMFPointer(0)
	serviceRequest.SetTMSI5G([4]uint8{0, 0, 0, 1})
	serviceRequest.TMSI5GS.SetLen(7)
	switch serviceType {
	case nasMessage.ServiceTypeMobileTerminatedServices:
		serviceRequest.AllowedPDUSessionStatus = new(nasType.AllowedPDUSessionStatus)
		serviceRequest.AllowedPDUSessionStatus.SetIei(nasMessage.ServiceRequestAllowedPDUSessionStatusType)
		serviceRequest.AllowedPDUSessionStatus.SetLen(2)
		serviceRequest.AllowedPDUSessionStatus.Buffer = []uint8{0x00, 0x08}
	case nasMessage.ServiceTypeData:
		serviceRequest.UplinkDataStatus = new(nasType.UplinkDataStatus)
		serviceRequest.UplinkDataStatus.SetIei(nasMessage.ServiceRequestUplinkDataStatusType)
		serviceRequest.UplinkDataStatus.SetLen(2)
		serviceRequest.UplinkDataStatus.Buffer = []uint8{0x00, 0x04}
	case nasMessage.ServiceTypeSignalling:
	}

	m.GmmMessage.ServiceRequest = serviceRequest
	return m
}
