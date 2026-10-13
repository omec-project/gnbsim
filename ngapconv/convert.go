// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-FileCopyrightText: 2019 Communication Service/Software Laboratory, National Chiao Tung University (free5gc.org)
// SPDX-License-Identifier: Apache-2.0

// Package ngapconv translates gNBSim's OpenAPI models and NGAP wire types.
package ngapconv

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/logger"
	"github.com/omec-project/ngap/v2/ngapType"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
)

func validBitString(bitString aper.BitString, bitLength int) bool {
	return bitString.BitLength == uint64(bitLength) && len(bitString.Bytes) == (bitLength+7)/8
}

func bitStringToHex(bitString *aper.BitString) string {
	hexString := hex.EncodeToString(bitString.Bytes)
	hexLen := (bitString.BitLength + 3) / 4
	return hexString[:hexLen]
}

func hexToBitString(hexString string, bitLength int) (bitString aper.BitString) {
	if bitLength <= 0 || len(hexString) != (bitLength+3)/4 {
		logger.NgapLog.Warnf("hexLen[%d] doesn't match bitLength[%d]", len(hexString), bitLength)
		return bitString
	}
	if len(hexString)%2 == 1 {
		hexString += "0"
	}
	bytes, err := hex.DecodeString(hexString)
	if err != nil || len(bytes) != (bitLength+7)/8 {
		logger.NgapLog.Warnf("Decode byteString failed: %+v", err)
		return bitString
	}
	bitString.Bytes = bytes
	bitString.BitLength = uint64(bitLength)
	if bitLength%8 != 0 {
		bitString.Bytes[len(bitString.Bytes)-1] &= byte(0xff) << uint(8-bitLength%8)
	}
	return bitString
}

// PlmnIdToModels converts an NGAP PLMN identity to an OpenAPI model.
func PlmnIdToModels(ngapPlmnID ngapType.PLMNIdentity) (models.PlmnId, error) {
	value := ngapPlmnID.Value
	if len(value) != 3 {
		return models.PlmnId{}, fmt.Errorf("invalid PLMNIdentity length %d", len(value))
	}
	hexString := hex.EncodeToString(value)
	mcc := []byte{hexString[1], hexString[0], hexString[3]}
	for _, digit := range mcc {
		if digit < '0' || digit > '9' {
			return models.PlmnId{}, fmt.Errorf("invalid MCC digit %q in PLMNIdentity %q", digit, hexString)
		}
	}
	plmn := models.PlmnId{Mcc: string(mcc)}
	if hexString[2] == 'f' {
		if hexString[5] < '0' || hexString[5] > '9' || hexString[4] < '0' || hexString[4] > '9' {
			return models.PlmnId{}, fmt.Errorf("invalid two-digit MNC in PLMNIdentity %q", hexString)
		}
		plmn.Mnc = string([]byte{hexString[5], hexString[4]})
	} else {
		mnc := []byte{hexString[2], hexString[5], hexString[4]}
		for _, digit := range mnc {
			if digit < '0' || digit > '9' {
				return models.PlmnId{}, fmt.Errorf("invalid three-digit MNC in PLMNIdentity %q", hexString)
			}
		}
		plmn.Mnc = string(mnc)
	}
	return plmn, nil
}

// PlmnIdToNgap converts an OpenAPI PLMN model to an NGAP identity.
func PlmnIdToNgap(plmn models.PlmnId) ngapType.PLMNIdentity {
	if len(plmn.Mcc) != 3 || (len(plmn.Mnc) != 2 && len(plmn.Mnc) != 3) {
		logger.NgapLog.Warnf("invalid PLMN lengths: MCC %q, MNC %q", plmn.Mcc, plmn.Mnc)
		return ngapType.PLMNIdentity{}
	}
	mcc, mnc := strings.Split(plmn.Mcc, ""), strings.Split(plmn.Mnc, "")
	var hexString string
	if len(plmn.Mnc) == 2 {
		hexString = mcc[1] + mcc[0] + "f" + mcc[2] + mnc[1] + mnc[0]
	} else {
		hexString = mcc[1] + mcc[0] + mnc[0] + mcc[2] + mnc[2] + mnc[1]
	}
	value, err := hex.DecodeString(hexString)
	if err != nil {
		logger.NgapLog.Warnf("decode plmn failed: %+v", err)
		return ngapType.PLMNIdentity{}
	}
	return ngapType.PLMNIdentity{Value: value}
}

// SNssaiToModels converts an NGAP S-NSSAI to an OpenAPI model.
func SNssaiToModels(ngapSNssai ngapType.SNSSAI) (models.Snssai, error) {
	if len(ngapSNssai.SST.Value) != 1 {
		return models.Snssai{}, fmt.Errorf("invalid S-NSSAI SST length %d", len(ngapSNssai.SST.Value))
	}
	snssai := models.Snssai{Sst: int32(ngapSNssai.SST.Value[0])}
	if ngapSNssai.SD != nil {
		if len(ngapSNssai.SD.Value) != 3 {
			return models.Snssai{}, fmt.Errorf("invalid S-NSSAI SD length %d", len(ngapSNssai.SD.Value))
		}
		snssai.Sd = openapi.PtrString(hex.EncodeToString(ngapSNssai.SD.Value))
	}
	return snssai, nil
}

// SNssaiToNgap converts an OpenAPI S-NSSAI model to NGAP.
func SNssaiToNgap(snssai models.Snssai) ngapType.SNSSAI {
	result := ngapType.SNSSAI{}
	result.SST.Value = []byte{byte(snssai.GetSst())}
	if snssai.GetSd() != "" {
		result.SD = new(ngapType.SD)
		value, err := hex.DecodeString(snssai.GetSd())
		if err != nil {
			logger.NgapLog.Warnf("decode snssai.sd failed: %+v", err)
		} else {
			result.SD.Value = value
		}
	}
	return result
}

// AmfIdToModels converts AMF region, set, and pointer bit strings to AMF ID.
func AmfIdToModels(regionID, setID, pointerID aper.BitString) string {
	regionHex := bitStringToHex(&regionID)
	tmp := []byte{setID.Bytes[0], (setID.Bytes[1] & 0xc0) | (pointerID.Bytes[0] >> 2)}
	return regionHex + hex.EncodeToString(tmp)
}

// RanIDToNgap converts gNBSim's RAN node identifier to NGAP.
func RanIDToNgap(ranID models.GlobalRanNodeId) ngapType.GlobalRANNodeID {
	var result ngapType.GlobalRANNodeID
	if ranID.GNbId != nil && ranID.GNbId.BitLength != 0 {
		bits := hexToBitString(ranID.GNbId.GNBValue, int(ranID.GNbId.BitLength))
		if !validBitString(bits, int(ranID.GNbId.BitLength)) {
			logger.NgapLog.Warnf("RanIDToNgap: invalid GNBID payload %q with bit length %d", ranID.GNbId.GNBValue, ranID.GNbId.BitLength)
			return result
		}
		result.Present = ngapType.GlobalRANNodeIDPresentGlobalGNBID
		result.GlobalGNBID = &ngapType.GlobalGNBID{PLMNIdentity: PlmnIdToNgap(ranID.PlmnId)}
		result.GlobalGNBID.GNBID.Present = ngapType.GNBIDPresentGNBID
		result.GlobalGNBID.GNBID.GNBID = &bits
		return result
	}
	if value := ranID.GetNgeNbId(); value != "" {
		global := &ngapType.GlobalNgENBID{PLMNIdentity: PlmnIdToNgap(ranID.PlmnId)}
		var bits aper.BitString
		switch {
		case strings.HasPrefix(value, "MacroNGeNB-"):
			bits = hexToBitString(value[11:], 18)
			if validBitString(bits, 18) {
				global.NgENBID.Present = ngapType.NgENBIDPresentMacroNgENBID
				global.NgENBID.MacroNgENBID = &bits
			}
		case strings.HasPrefix(value, "SMacroNGeNB-"):
			bits = hexToBitString(value[12:], 20)
			if validBitString(bits, 20) {
				global.NgENBID.Present = ngapType.NgENBIDPresentShortMacroNgENBID
				global.NgENBID.ShortMacroNgENBID = &bits
			}
		case strings.HasPrefix(value, "LMacroNGeNB-"):
			bits = hexToBitString(value[12:], 21)
			if validBitString(bits, 21) {
				global.NgENBID.Present = ngapType.NgENBIDPresentLongMacroNgENBID
				global.NgENBID.LongMacroNgENBID = &bits
			}
		}
		if global.NgENBID.Present != 0 {
			result.Present = ngapType.GlobalRANNodeIDPresentGlobalNgENBID
			result.GlobalNgENBID = global
			return result
		}
		logger.NgapLog.Warnf("RanIDToNgap: invalid NgENBID payload %q", value)
		return result
	}
	if value := ranID.GetN3IwfId(); value != "" {
		bits := hexToBitString(value, len(value)*4)
		if validBitString(bits, len(value)*4) {
			result.Present = ngapType.GlobalRANNodeIDPresentGlobalN3IWFID
			result.GlobalN3IWFID = &ngapType.GlobalN3IWFID{PLMNIdentity: PlmnIdToNgap(ranID.PlmnId)}
			result.GlobalN3IWFID.N3IWFID.Present = ngapType.N3IWFIDPresentN3IWFID
			result.GlobalN3IWFID.N3IWFID.N3IWFID = &bits
			return result
		}
		logger.NgapLog.Warnf("RanIDToNgap: invalid N3IWFID payload %q", value)
		return result
	}
	logger.NgapLog.Warnf("RanIDToNgap: no supported RAN node identifier found for PLMN %+v", ranID.PlmnId)
	return result
}

// IPAddressToString converts a transport-layer address into IPv4 and IPv6 strings.
func IPAddressToString(address ngapType.TransportLayerAddress) (ipv4, ipv6 string) {
	bits := address.Value
	switch bits.BitLength {
	case 32:
		return net.IPv4(bits.Bytes[0], bits.Bytes[1], bits.Bytes[2], bits.Bytes[3]).String(), ""
	case 128:
		return "", net.IP(bits.Bytes).String()
	case 160:
		return net.IPv4(bits.Bytes[0], bits.Bytes[1], bits.Bytes[2], bits.Bytes[3]).String(), net.IP(bits.Bytes[4:]).String()
	}
	return "", ""
}

// IPAddressToNgap converts IPv4 and/or IPv6 strings to a transport-layer address.
func IPAddressToNgap(ipv4, ipv6 string) ngapType.TransportLayerAddress {
	var bytes []byte
	var bitLength uint64
	if ipv4 != "" {
		parsed := net.ParseIP(ipv4)
		if parsed == nil || parsed.To4() == nil {
			logger.NgapLog.Warnf("IPAddressToNgap: invalid IPv4 address %q", ipv4)
			return ngapType.TransportLayerAddress{}
		}
		bytes = append(bytes, parsed.To4()...)
		bitLength = 32
	}
	if ipv6 != "" {
		parsed := net.ParseIP(ipv6)
		if parsed == nil || parsed.To16() == nil {
			logger.NgapLog.Warnf("IPAddressToNgap: invalid IPv6 address %q", ipv6)
			return ngapType.TransportLayerAddress{}
		}
		bytes = append(bytes, parsed.To16()...)
		bitLength += 128
	}
	if bitLength == 0 {
		logger.NgapLog.Warnln("IPAddressToNgap: Both ipv4 and ipv6 are empty string")
	}
	return ngapType.TransportLayerAddress{Value: aper.BitString{Bytes: bytes, BitLength: bitLength}}
}
