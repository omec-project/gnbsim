// SPDX-FileCopyrightText: 2026 Intel Corporation
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

// Package nasconv translates between OpenAPI models and NAS protocol types.
//
// It is owned by gNBSim so the NAS protocol library can remain independent of
// OpenAPI-generated SBI models.
package nasconv

import (
	"encoding/hex"
	"strconv"

	"github.com/omec-project/nas/v2/logger"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/nasType"
	"github.com/omec-project/openapi/v2/models"
)

func GutiToString(buf []byte) (models.Guami, string) {
	if len(buf) != 11 {
		logger.ConvertLog.Errorf("invalid GUTI buffer length: %d", len(buf))
		return models.Guami{}, ""
	}
	plmnID := plmnIDToString(buf[1:4])
	guami := models.Guami{
		PlmnId: models.PlmnIdNid{Mcc: plmnID[:3], Mnc: plmnID[3:]},
		AmfId:  hex.EncodeToString(buf[4:7]),
	}
	return guami, plmnID + guami.AmfId + hex.EncodeToString(buf[7:])
}

func GutiToNas(guti string) nasType.GUTI5G {
	result := nasType.GUTI5G{}
	result.SetLen(11)
	result.SetSpare(0)
	result.SetSpare2(15)
	result.SetTypeOfIdentity(nasMessage.MobileIdentity5GSType5gGuti)

	digit := func(index int) int {
		value, err := strconv.Atoi(string(guti[index]))
		if err != nil {
			logger.ConvertLog.Warnf("parse GUTI digit: %v", err)
		}
		return value
	}
	mcc1, mcc2, mcc3 := digit(0), digit(1), digit(2)
	mnc1, mnc2, mnc3 := digit(3), digit(4), 0x0f
	amfID, tmsi := "", ""
	if len(guti) == 20 {
		mnc3 = digit(5)
		amfID, tmsi = guti[6:12], guti[12:]
	} else {
		amfID, tmsi = guti[5:11], guti[11:]
	}
	result.SetMCCDigit1(uint8(mcc1))
	result.SetMCCDigit2(uint8(mcc2))
	result.SetMCCDigit3(uint8(mcc3))
	result.SetMNCDigit1(uint8(mnc1))
	result.SetMNCDigit2(uint8(mnc2))
	result.SetMNCDigit3(uint8(mnc3))
	region, set, pointer := amfIDToNas(amfID)
	result.SetAMFRegionID(region)
	result.SetAMFSetID(set)
	result.SetAMFPointer(pointer)
	if value, err := hex.DecodeString(tmsi); err != nil {
		logger.ConvertLog.Warnf("decode TMSI: %v", err)
	} else {
		copy(result.Octet[7:11], value)
	}
	return result
}

func SpareHalfOctetAndNgksiToModels(ngKsi nasType.SpareHalfOctetAndNgksi) models.NgKsi {
	result := models.NgKsi{Ksi: int32(ngKsi.GetNasKeySetIdentifiler())}
	switch ngKsi.GetTSC() {
	case nasMessage.TypeOfSecurityContextFlagNative:
		result.Tsc = models.SCTYPE_NATIVE
	case nasMessage.TypeOfSecurityContextFlagMapped:
		result.Tsc = models.SCTYPE_MAPPED
	}
	return result
}

func PDUSessionTypeToModels(nasPDUSessionType uint8) models.PduSessionType {
	switch nasPDUSessionType {
	case nasMessage.PDUSessionTypeIPv4:
		return models.PDUSESSIONTYPE_IPV4
	case nasMessage.PDUSessionTypeIPv6:
		return models.PDUSESSIONTYPE_IPV6
	case nasMessage.PDUSessionTypeIPv4IPv6:
		return models.PDUSESSIONTYPE_IPV4_V6
	case nasMessage.PDUSessionTypeUnstructured:
		return models.PDUSESSIONTYPE_UNSTRUCTURED
	case nasMessage.PDUSessionTypeEthernet:
		return models.PDUSESSIONTYPE_ETHERNET
	default:
		return ""
	}
}

func amfIDToNas(amfID string) (uint8, uint16, uint8) {
	value, err := hex.DecodeString(amfID)
	if err != nil {
		logger.ConvertLog.Errorf("decode AMF ID: %v", err)
	}
	return value[0], uint16(value[1])<<2 + (uint16(value[2])&0x00c0)>>6, value[2] & 0x3f
}

func plmnIDToString(buf []byte) string {
	if len(buf) < 3 {
		logger.ConvertLog.Errorf("invalid PLMN ID buffer length: %d", len(buf))
		return ""
	}
	mcc1, mcc2, mcc3 := buf[0]&0x0f, (buf[0]&0xf0)>>4, buf[1]&0x0f
	mnc1, mnc2, mnc3 := buf[2]&0x0f, (buf[2]&0xf0)>>4, (buf[1]&0xf0)>>4
	value := hex.EncodeToString([]byte{(mcc1 << 4) | mcc2, (mcc3 << 4) | mnc1, (mnc2 << 4) | mnc3})
	if value[5] == 'f' {
		return value[:5]
	}
	return value
}
