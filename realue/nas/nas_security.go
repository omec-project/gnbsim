// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"crypto/subtle"
	"fmt"
	"sync"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/security"
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
		switch msg.SecurityHeaderType {
		case nas.SecurityHeaderTypeIntegrityProtected:
			ue.Log.Debugln("security header type: Integrity Protected")
		case nas.SecurityHeaderTypeIntegrityProtectedAndCiphered:
			ue.Log.Debugln("security header type: Integrity Protected And Ciphered")
			needCiphering = true
		case nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext:
			ue.Log.Debugln("security header type: Integrity Protected With New 5G Security Context")
			ue.ULCount.Set(0, 0)
			ue.DLCount.Set(0, 0)
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			ue.Log.Debugln("security header type: Integrity Protected With New 5G Security Context")
			ue.ULCount.Set(0, 0)
			ue.DLCount.Set(0, 0)
			needCiphering = true
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeaderType)
		}

		payload, err = msg.PlainNasEncode()
		if err != nil {
			return nil, fmt.Errorf("plain nas encode failed: %+v", err)
		}

		if needCiphering {
			ue.Log.Debugf("Encrypt NAS message (algorithm: %+v, DLCount: 0x%0x)", ue.CipheringAlg, ue.DLCount.Get())
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
		msgSecurityHeader := []byte{msg.ProtocolDiscriminator, msg.SecurityHeaderType}
		payload = append(msgSecurityHeader, payload[:]...)

		// Increase UL Count
		ue.ULCount.AddOne()
	}
	return payload, err
}

// minSecurityProtectedLen is the length of the security header of a security protected 5GS NAS
// message: EPD (1), security header type (1), MAC (4) and sequence number (1).
const minSecurityProtectedLen = 7

// 5GMM causes after which a REGISTRATION REJECT / SERVICE REJECT must be integrity protected
// (TS 24.501 clause 4.4.4.2 e) and g)).
const (
	cause5GMMCAGNotAuthorized  uint8 = 76
	cause5GMMPlmnNotAllowedLoc uint8 = 78
)

// NASDecode decodes a downlink NAS message. Once the secure exchange of NAS messages has been
// established, a message that is not integrity protected (other than the exceptions listed in
// TS 24.501 clause 4.4.4.2) or that fails the integrity check is discarded with an error, and
// the downlink NAS COUNT is left unchanged.
func NASDecode(ue *realuectx.RealUe, securityHeaderType uint8, payload []byte) (msg *nas.Message, err error) {
	decodeMutex.Lock()
	defer decodeMutex.Unlock()

	if ue == nil {
		return nil, fmt.Errorf("amfUe is nil")
	}
	if payload == nil {
		return nil, fmt.Errorf("nas payload is empty")
	}
	if len(payload) < 2 {
		return nil, fmt.Errorf("nas payload too short: %d bytes", len(payload))
	}

	msg = new(nas.Message)
	msg.SecurityHeaderType = nas.GetSecurityHeaderType(payload)
	if securityHeaderType == nas.SecurityHeaderTypePlainNas {
		epd := payload[0]
		if err = msg.PlainNasDecode(&payload); err != nil {
			return msg, err
		}
		if ue.NasSecurityEstablished && !plainNasAllowed(epd, msg) {
			return nil, fmt.Errorf("discarding plain NAS message received after the secure exchange of NAS messages was established")
		}
		return msg, nil
	} else if ue.IntegrityAlg == security.AlgIntegrity128NIA0 {
		if len(payload) < 4 {
			return nil, fmt.Errorf("nas payload too short: %d bytes", len(payload))
		}
		ue.Log.Debugln("decode payload is", payload)
		// remove header
		payload = payload[3:]

		if err = security.NASEncrypt(ue.CipheringAlg, ue.KnasEnc, ue.DLCount.Get(), security.Bearer3GPP,
			security.DirectionDownlink, payload); err != nil {
			return nil, err
		}

		err = msg.PlainNasDecode(&payload)
		return msg, err
	} else { // Security protected NAS message
		if len(payload) < minSecurityProtectedLen {
			return nil, fmt.Errorf("security protected nas payload too short: %d bytes", len(payload))
		}
		sequenceNumber := payload[6]
		receivedMac32 := payload[2:6]
		// remove security Header except for sequece Number
		payload = payload[6:]

		// the NAS COUNT is only updated once the integrity check has passed (TS 24.501 clause
		// 4.4.3.3), so the verification works on a copy
		dlCount := ue.DLCount

		// a security protected NAS message must be integrity protected, and ciphering is optional
		ciphered := false
		switch msg.SecurityHeaderType {
		case nas.SecurityHeaderTypeIntegrityProtected:
			ue.Log.Debugln("security header type: Integrity Protected")
		case nas.SecurityHeaderTypeIntegrityProtectedAndCiphered:
			ue.Log.Debugln("security header type: Integrity Protected And Ciphered")
			ciphered = true
		case nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext:
			ue.Log.Debugln("security Header Type Integrity Protected With New 5g Nas Security Context")
			dlCount.Set(0, 0)
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			ue.Log.Debugln("security header type: Integrity Protected And Ciphered With New 5G Security Context")
			ciphered = true
			dlCount.Set(0, 0)
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeaderType)
		}
		// Caculate dl count
		if dlCount.SQN() > sequenceNumber {
			dlCount.SetOverflow(dlCount.Overflow() + 1)
		}
		dlCount.SetSQN(sequenceNumber)

		ue.Log.Debugf("calculate NAS MAC (algorithm: %+v, DLCount: 0x%0x)", ue.IntegrityAlg, dlCount.Get())

		mac32, errNas := security.NASMacCalculate(ue.IntegrityAlg, ue.KnasInt, dlCount.Get(), security.Bearer3GPP,
			security.DirectionDownlink, payload)
		if errNas != nil {
			return nil, errNas
		}
		if subtle.ConstantTimeCompare(mac32, receivedMac32) != 1 {
			ue.Log.Warnf("NAS MAC verification failed(0x%x != 0x%x), discarding message", mac32, receivedMac32)
			return nil, fmt.Errorf("NAS MAC verification failed")
		}
		ue.Log.Debugf("cmac value: 0x%x", mac32)
		ue.DLCount = dlCount
		ue.NasSecurityEstablished = true

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

// plainNasAllowed reports whether a plain (not integrity protected) 5GMM message may still be
// processed after the secure exchange of NAS messages has been established, per the list in
// TS 24.501 clause 4.4.4.2. IDENTITY REQUEST is accepted without checking that SUCI is the
// requested identity, and DEREGISTRATION ACCEPT without telling switch-off apart.
func plainNasAllowed(epd uint8, msg *nas.Message) bool {
	if epd != nasMessage.Epd5GSMobilityManagementMessage {
		return false
	}
	switch msg.GmmHeader.GetMessageType() {
	case nas.MsgTypeIdentityRequest,
		nas.MsgTypeAuthenticationRequest,
		nas.MsgTypeAuthenticationResult,
		nas.MsgTypeAuthenticationReject,
		nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration:
		return true
	case nas.MsgTypeRegistrationReject:
		return msg.RegistrationReject != nil && !integrityRequiredCause(msg.RegistrationReject.GetCauseValue())
	case nas.MsgTypeServiceReject:
		return msg.ServiceReject != nil && !integrityRequiredCause(msg.ServiceReject.GetCauseValue())
	default:
		return false
	}
}

func integrityRequiredCause(cause uint8) bool {
	return cause == cause5GMMCAGNotAuthorized || cause == cause5GMMPlmnNotAllowedLoc
}
