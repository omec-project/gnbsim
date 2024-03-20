// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"fmt"
	"reflect"
	"sync"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/security"
	"github.com/omec-project/ngap/ngapType"
)

var (
	decodeMutex sync.Mutex
	encodeMutex sync.Mutex
)

func EncodeNasPduWithSecurity(ue *realuectx.RealUe, pdu []byte, securityHeaderType uint8,
	securityContextAvailable bool,
) ([]byte, error) {
	m := nas.NewMessage()
	err := m.PlainNasDecode(&pdu)
	if err != nil {
		return nil, err
	}
	m.SecurityHeader = nas.SecurityHeader{
		ProtocolDiscriminator: nasMessage.Epd5GSMobilityManagementMessage,
		SecurityHeaderType:    securityHeaderType,
	}
	return NASEncode(ue, m, securityContextAvailable)
}

func GetNasPdu(ue *realuectx.RealUe, msg *ngapType.DownlinkNASTransport) (m *nas.Message) {
	for _, ie := range msg.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDNASPDU {
			pkg := []byte(ie.Value.NASPDU.Value)
			m, err := NASDecode(ue, nas.GetSecurityHeaderType(pkg), pkg)
			if err != nil {
				return nil
			}
			return m
		}
	}
	return nil
}

func GetNasPduSetupRequest(ue *realuectx.RealUe, msg *ngapType.PDUSessionResourceSetupRequest) (m *nas.Message) {
	for _, ie := range msg.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDPDUSessionResourceSetupListSUReq {
			x := ie.Value.PDUSessionResourceSetupListSUReq
			for _, ie1 := range x.List {
				if ie1.PDUSessionNASPDU != nil {
					fmt.Println("Found NAS PDU inside ResourceSEtupList")
					pkg := []byte(ie1.PDUSessionNASPDU.Value)
					m, err := NASDecode(ue, nas.GetSecurityHeaderType(pkg), pkg)
					fmt.Println("UE address - ", m.GmmMessage.DLNASTransport.Ipaddr)
					if err != nil {
						return nil
					}
					return m
				}
			}
		}
	}
	return nil
}

func NASEncode(ue *realuectx.RealUe, msg *nas.Message, securityContextAvailable bool) (
	payload []byte, err error,
) {
	encodeMutex.Lock()
	defer encodeMutex.Unlock()

	if ue == nil {
		return nil, fmt.Errorf("amfUe is nil")
	}
	if msg == nil {
		return nil, fmt.Errorf("nas message is empty")
	}

	if !securityContextAvailable {
		return msg.PlainNasEncode()
	} else {
		needCiphering := false
		switch msg.SecurityHeader.SecurityHeaderType {
		case nas.SecurityHeaderTypeIntegrityProtected:
			ue.Log.Debugln("Security header type: Integrity Protected")
		case nas.SecurityHeaderTypeIntegrityProtectedAndCiphered:
			ue.Log.Debugln("Security header type: Integrity Protected And Ciphered")
			needCiphering = true
		case nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext:
			ue.Log.Debugln("Security header type: Integrity Protected With New 5G Security Context")
			ue.ULCount.Set(0, 0)
			ue.DLCount.Set(0, 0)
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			ue.Log.Debugln("Security header type: Integrity Protected With New 5G Security Context")
			ue.ULCount.Set(0, 0)
			ue.DLCount.Set(0, 0)
			needCiphering = true
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeader.SecurityHeaderType)
		}

		payload, err = msg.PlainNasEncode()
		if err != nil {
			return nil, fmt.Errorf("plain nas encode failed: %+v", err)
		}

		if needCiphering {
			ue.Log.Debugf("Encrypt NAS message (algorithm: %+v, DLCount: 0x%0x)", ue.CipheringAlg, ue.DLCount.Get())
			ue.Log.Tracef("NAS ciphering key: %0x", ue.KnasEnc)
			// TODO: Support for ue has nas connection in both accessType
			if err = security.NASEncrypt(ue.CipheringAlg, ue.KnasEnc, ue.ULCount.Get(), security.Bearer3GPP,
				security.DirectionUplink, payload); err != nil {
				return nil, fmt.Errorf("encrypt error: %+v", err)
			}
		}
		// add sequence number
		payload = append([]byte{ue.ULCount.SQN()}, payload[:]...)

		var mac32 []byte
		mac32, err = security.NASMacCalculate(ue.IntegrityAlg, ue.KnasInt, ue.ULCount.Get(),
			security.Bearer3GPP, security.DirectionUplink, payload)
		if err != nil {
			return nil, fmt.Errorf("nas mac calcuate failed: %+v", err)
		}

		// Add mac value
		payload = append(mac32, payload[:]...)
		// Add EPD and Security Type
		msgSecurityHeader := []byte{msg.SecurityHeader.ProtocolDiscriminator, msg.SecurityHeader.SecurityHeaderType}
		payload = append(msgSecurityHeader, payload[:]...)

		// Increase UL Count
		ue.ULCount.AddOne()
	}
	return payload, err
}

func NASDecode(ue *realuectx.RealUe, securityHeaderType uint8, payload []byte) (msg *nas.Message, err error) {
	decodeMutex.Lock()
	defer decodeMutex.Unlock()

	if ue == nil {
		return nil, fmt.Errorf("amfUe is nil")
	}
	if payload == nil {
		return nil, fmt.Errorf("nas payload is empty")
	}

	msg = new(nas.Message)
	msg.SecurityHeaderType = nas.GetSecurityHeaderType(payload)
	if securityHeaderType == nas.SecurityHeaderTypePlainNas {
		err = msg.PlainNasDecode(&payload)
		return msg, err
	} else if ue.IntegrityAlg == security.AlgIntegrity128NIA0 {
		ue.Log.Debugln("decode payload is ", payload)
		// remove header
		payload = payload[3:]

		if err = security.NASEncrypt(ue.CipheringAlg, ue.KnasEnc, ue.DLCount.Get(), security.Bearer3GPP,
			security.DirectionDownlink, payload); err != nil {
			return nil, err
		}

		err = msg.PlainNasDecode(&payload)
		return msg, err
	} else { // Security protected NAS message
		securityHeader := payload[0:6]
		sequenceNumber := payload[6]
		receivedMac32 := securityHeader[2:]
		// remove security Header except for sequece Number
		payload = payload[6:]

		// a security protected NAS message must be integrity protected, and ciphering is optional
		ciphered := false
		switch msg.SecurityHeaderType {
		case nas.SecurityHeaderTypeIntegrityProtected:
			ue.Log.Debugln("Security header type: Integrity Protected")
		case nas.SecurityHeaderTypeIntegrityProtectedAndCiphered:
			ue.Log.Debugln("Security header type: Integrity Protected And Ciphered")
			ciphered = true
		case nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext:
			ue.Log.Debugln("Security Header Type Integrity Protected With New 5g Nas Security Context")
			ue.DLCount.Set(0, 0)
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			ue.Log.Debugln("Security header type: Integrity Protected And Ciphered With New 5G Security Context")
			ciphered = true
			ue.DLCount.Set(0, 0)
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeader.SecurityHeaderType)
		}
		// Caculate ul count
		if ue.DLCount.SQN() > sequenceNumber {
			ue.DLCount.SetOverflow(ue.DLCount.Overflow() + 1)
		}
		ue.DLCount.SetSQN(sequenceNumber)

		ue.Log.Infof("Calculate NAS MAC (algorithm: %+v, DLCount: 0x%0x)", ue.IntegrityAlg, ue.DLCount.Get())
		ue.Log.Infof("NAS integrity key: %0x", ue.KnasInt)

		mac32, errNas := security.NASMacCalculate(ue.IntegrityAlg, ue.KnasInt, ue.DLCount.Get(), security.Bearer3GPP,
			security.DirectionDownlink, payload)
		if errNas != nil {
			return nil, errNas
		}
		if !reflect.DeepEqual(mac32, receivedMac32) {
			ue.Log.Warnf("NAS MAC verification failed(0x%x != 0x%x)", mac32, receivedMac32)
		} else {
			ue.Log.Infof("cmac value: 0x%x", mac32)
		}

		// remove sequece Number
		payload = payload[1:]
		if ciphered {
			if err = security.NASEncrypt(ue.CipheringAlg, ue.KnasEnc, ue.DLCount.Get(), security.Bearer3GPP,
				security.DirectionDownlink, payload); err != nil {
				return nil, err
			}
		}

		err = msg.PlainNasDecode(&payload)
		return msg, err
	}
}
