// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"math"
	"sync"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/gnbsim/realue/nasconv"
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
			// The COUNTs are not touched here: the uplink COUNT starts over once, when
			// NASDecode takes the new security context into use, so a response to a
			// retransmitted SECURITY MODE COMMAND goes out with the next COUNT instead of
			// reusing one under the same keys.
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			ue.Log.Debugln("security header type: Integrity Protected And Ciphered With New 5G Security Context")
			needCiphering = true
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeaderType)
		}

		// The uplink COUNT must not wrap around under the same keys, which would reuse a COUNT: the
		// message is not sent (TS 24.501 clause 4.4.3.5). Null integrity allows the wrap around.
		if ue.IntegrityAlg != security.AlgIntegrity128NIA0 && ue.ULCount.Get() == maxNasCount {
			return nil, fmt.Errorf("uplink NAS COUNT exhausted, a new security context is needed")
		}

		payload, err = msg.PlainNasEncode()
		if err != nil {
			return nil, fmt.Errorf("plain nas encode failed: %+v", err)
		}

		if needCiphering {
			ue.Log.Debugf("Encrypt NAS message (algorithm: %+v, ULCount: 0x%0x)", ue.CipheringAlg, ue.ULCount.Get())
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

// ErrNasDiscarded is wrapped by the errors NASDecode returns for a message that the UE has to
// discard (TS 24.501 clause 4.4.4.2) as opposed to one it fails to decode. Discarding a message
// does not fail the ongoing procedure, so callers skip the PDU when errors.Is matches.
var ErrNasDiscarded = errors.New("NAS message discarded")

// minSecurityProtectedLen is the length of the security header of a security protected 5GS NAS
// message: EPD (1), security header type (1), MAC (4) and sequence number (1).
const minSecurityProtectedLen = 7

// maxNasCount is the largest NAS COUNT, which is 24 bits wide.
const maxNasCount = 0x00ffffff

// 5GMM causes after which a REGISTRATION REJECT / SERVICE REJECT must be integrity protected
// (TS 24.501 clause 4.4.4.2 e) and g)).
const (
	cause5GMMCAGNotAuthorized  uint8 = 76
	cause5GMMPlmnNotAllowedLoc uint8 = 78
)

// NASDecode decodes a downlink NAS message. Once the secure exchange of NAS messages has been
// established, a message that is not integrity protected (other than the exceptions listed in
// TS 24.501 clause 4.4.4.2) or that fails the integrity check is discarded with an error that
// wraps ErrNasDiscarded, and the downlink NAS COUNT is left unchanged. The security header type
// is not integrity protected, so a "new 5G NAS security context" type is only accepted for a
// SECURITY MODE COMMAND, and only that one that takes freshly derived keys into use resets the
// COUNT. It is verified under those pending keys first, and under the keys and COUNT in use if
// they do not authenticate it (a retransmission); every other message is verified under the keys
// and COUNT in use. A message that estimates to an already accepted COUNT is a replay
// and is discarded. A plain SECURITY MODE COMMAND is never processed.
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
		// Only the messages of TS 24.501 clause 4.4.4.2 are processed without integrity protection,
		// also before the secure exchange of NAS messages is established. A SECURITY MODE COMMAND
		// is not among them: without a successful integrity check it must not be answered.
		// The header says whether the message may be processed at all, so a message that may not
		// is discarded before its body is decoded: a malformed body must not make an
		// unauthenticated message fail the procedure instead.
		if !plainNasTypeAllowed(ue, payload) {
			return nil, fmt.Errorf("%w: plain NAS message that must be integrity protected", ErrNasDiscarded)
		}
		if err = msg.PlainNasDecode(&payload); err != nil {
			if ue.NasSecurityEstablished {
				return nil, fmt.Errorf("%w: undecodable plain NAS message received after the secure exchange of NAS messages was established: %v",
					ErrNasDiscarded, err)
			}
			return msg, err
		}
		if !plainNasContentAllowed(msg) {
			return nil, fmt.Errorf("%w: plain NAS message that must be integrity protected", ErrNasDiscarded)
		}
		return msg, nil
	} else { // Security protected NAS message
		if len(payload) < minSecurityProtectedLen {
			return nil, fmt.Errorf("security protected nas payload too short: %d bytes", len(payload))
		}
		sequenceNumber := payload[6]
		receivedMac32 := payload[2:6]
		// remove security Header except for sequece Number
		payload = payload[6:]

		// the NAS COUNT is only updated once the integrity check has passed (TS 24.501 clause
		// 4.4.3.3), so the verification works on copies. candidates lists the security contexts the
		// message may belong to, each with its own keys and COUNT.
		candidates := []nasCandidate{{
			knasInt: ue.KnasInt, dlCount: ue.DLCount, countInUse: ue.DLCountInUse,
			usable: ue.NasKeysDerived || ue.IntegrityAlg == security.AlgIntegrity128NIA0,
		}}

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
			// This type is only used for the SECURITY MODE COMMAND (TS 24.501 table 9.3.1).
			if !isSecurityModeCommand(payload) {
				return nil, fmt.Errorf("%w: downlink NAS message other than a SECURITY MODE COMMAND with a new 5G NAS security context",
					ErrNasDiscarded)
			}
			// The security header type is not covered by the MAC, so it cannot be allowed to switch
			// keys by itself. The command is verified under the pending keys, which are tried first,
			// and only one that passes takes them into use, which starts their COUNT. Pending keys
			// can be staged by an unprotected AUTHENTICATION REQUEST at any time, so they must not
			// hide the context in use: a retransmission of an accepted command (TS 24.501 5.4.2.4)
			// has a COUNT of its own (4.4.3.1) and is verified under the keys and COUNT in use, like
			// any other message, when the pending keys do not authenticate it.
			if pendingInt, pendingDL, pendingInUse, ok := ue.PendingNasKeys(); ok {
				candidates = append([]nasCandidate{{
					knasInt: pendingInt, dlCount: pendingDL, countInUse: pendingInUse, usable: true, pending: true,
				}}, candidates...)
			}
		case nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext:
			// the SECURITY MODE COMMAND is sent unciphered; this type is only used by the UE for the
			// SECURITY MODE COMPLETE (TS 24.501 5.4.2.2, table 9.3.1)
			return nil, fmt.Errorf("%w: downlink NAS message ciphered with a new 5G NAS security context",
				ErrNasDiscarded)
		default:
			return nil, fmt.Errorf("wrong security header type: 0x%0x", msg.SecurityHeaderType)
		}

		nullIntegrity := ue.IntegrityAlg == security.AlgIntegrity128NIA0
		var accepted *nasCandidate
		var discardErr error
		for i := range candidates {
			c := &candidates[i]
			// Until NAS keys have been derived the integrity key is all zero, which anyone can use to
			// compute a MAC: with it nothing is authenticated. Null integrity has no MAC to forge.
			if !c.usable {
				discardErr = fmt.Errorf("%w: security protected NAS message received before NAS keys were derived",
					ErrNasDiscarded)
				continue
			}
			// the COUNT last accepted in the context the message is checked against
			lastCount := c.dlCount.Get()
			// Caculate dl count
			if c.dlCount.SQN() > sequenceNumber {
				// The COUNT must not wrap around under the same keys: it would estimate to a COUNT that
				// was accepted before, and a recorded message would verify again (TS 24.501 clause
				// 4.4.3.5). Only null integrity, which has nothing to verify, allows it.
				if !nullIntegrity && c.dlCount.Overflow() == math.MaxUint16 {
					discardErr = fmt.Errorf("%w: DL COUNT exhausted, a new security context is needed", ErrNasDiscarded)
					continue
				}
				c.dlCount.SetOverflow(c.dlCount.Overflow() + 1)
			}
			c.dlCount.SetSQN(sequenceNumber)

			// A COUNT is accepted at most once, so a message that estimates to the COUNT of the last
			// accepted one is a replay (TS 24.501 4.4.3.2). That does not apply to null integrity.
			if !nullIntegrity && c.countInUse && c.dlCount.Get() == lastCount {
				discardErr = fmt.Errorf("%w: replayed NAS message (DL COUNT 0x%x already accepted)",
					ErrNasDiscarded, c.dlCount.Get())
				continue
			}

			// NIA0 is the null integrity algorithm: there is no MAC to verify, and since nothing was
			// integrity checked it does not establish the secure exchange of NAS messages either
			if !nullIntegrity {
				ue.Log.Debugf("calculate NAS MAC (algorithm: %+v, DLCount: 0x%0x)", ue.IntegrityAlg, c.dlCount.Get())

				mac32, errNas := security.NASMacCalculate(ue.IntegrityAlg, c.knasInt, c.dlCount.Get(), security.Bearer3GPP,
					security.DirectionDownlink, payload)
				if errNas != nil {
					return nil, errNas
				}
				if subtle.ConstantTimeCompare(mac32, receivedMac32) != 1 {
					ue.Log.Warnf("NAS MAC verification failed(0x%x != 0x%x)", mac32, receivedMac32)
					discardErr = fmt.Errorf("%w: NAS MAC verification failed", ErrNasDiscarded)
					continue
				}
				ue.Log.Debugf("cmac value: 0x%x", mac32)
				ue.NasSecurityEstablished = true
			}
			accepted = c
			break
		}
		if accepted == nil {
			return nil, discardErr
		}

		// remove sequece Number
		payload = payload[1:]
		if accepted.pending {
			// The SECURITY MODE COMMAND is not ciphered. The key set identifier of the new context is
			// taken from it, as the one in the unprotected AUTHENTICATION REQUEST is not covered by
			// any MAC (TS 24.501 clause 5.4.2.2: the command includes the ngKSI of the new context).
			if err = msg.PlainNasDecode(&payload); err != nil {
				return msg, err
			}
			if msg.SecurityModeCommand == nil {
				return nil, fmt.Errorf("security mode command not decoded")
			}
			ue.ActivatePendingNasContext(accepted.dlCount,
				nasconv.SpareHalfOctetAndNgksiToModels(msg.SecurityModeCommand.SpareHalfOctetAndNgksi))
			return msg, nil
		}
		ue.DLCount = accepted.dlCount
		ue.DLCountInUse = true
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

// nasCandidate is a security context a downlink security protected message is verified against:
// its integrity key, and the downlink COUNT state (the COUNT last accepted in it and whether one
// was). pending marks keys that a SECURITY MODE COMMAND takes into use once it passes; usable is
// false for a context whose keys cannot authenticate anything.
type nasCandidate struct {
	knasInt    [16]uint8
	dlCount    security.Count
	countInUse bool
	usable     bool
	pending    bool
}

// isSecurityModeCommand reports whether payload, a sequence number followed by a plain 5GMM
// message, carries a SECURITY MODE COMMAND. It only looks at the header, as the message is not
// integrity checked yet.
func isSecurityModeCommand(payload []byte) bool {
	return len(payload) >= 4 && payload[1] == nasMessage.Epd5GSMobilityManagementMessage &&
		payload[3] == nas.MsgTypeSecurityModeCommand
}

// plainNasTypeAllowed reports whether a plain (not integrity protected) 5GMM message may be
// processed, per the list in TS 24.501 clause 4.4.4.2. Everything else needs integrity protection.
// It only looks at the header (EPD, security header type and message type) and at the octet the
// condition of the entry depends on (the identity type of an IDENTITY REQUEST, the 5GMM cause of a
// REGISTRATION REJECT or SERVICE REJECT), which must be present. Deciding on these octets before
// the body is decoded means a malformed body cannot turn a message that must be discarded into a
// decode error. plainNasContentAllowed repeats the conditions on the decoded message.
// DEREGISTRATION ACCEPT is allowed only while a deregistration request without "switch off" is
// pending.
func plainNasTypeAllowed(ue *realuectx.RealUe, payload []byte) bool {
	if len(payload) < 3 || payload[0] != nasMessage.Epd5GSMobilityManagementMessage {
		return false
	}
	switch payload[2] {
	case nas.MsgTypeAuthenticationRequest,
		nas.MsgTypeAuthenticationResult,
		nas.MsgTypeAuthenticationReject:
		return true
	case nas.MsgTypeIdentityRequest:
		// octet 4: spare half octet, type of identity (TS 24.501 clause 8.2.21)
		return len(payload) >= 4 && payload[3]&0x07 == nasMessage.MobileIdentity5GSTypeSuci
	case nas.MsgTypeRegistrationReject, // 5GMM cause is octet 4
		nas.MsgTypeServiceReject:
		return len(payload) >= 4 && !integrityRequiredCause(payload[3])
	case nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration:
		return ue.NonSwitchOffDeregPending
	default:
		return false
	}
}

// plainNasContentAllowed checks the conditions of TS 24.501 clause 4.4.4.2 that depend on the
// decoded message: IDENTITY REQUEST only if SUCI is the requested identity, and REGISTRATION
// REJECT / SERVICE REJECT only if the 5GMM cause is not #76 or #78.
func plainNasContentAllowed(msg *nas.Message) bool {
	switch msg.GmmHeader.GetMessageType() {
	case nas.MsgTypeIdentityRequest:
		return msg.IdentityRequest != nil &&
			msg.IdentityRequest.GetTypeOfIdentity() == nasMessage.MobileIdentity5GSTypeSuci
	case nas.MsgTypeRegistrationReject:
		return msg.RegistrationReject != nil && !integrityRequiredCause(msg.RegistrationReject.GetCauseValue())
	case nas.MsgTypeServiceReject:
		return msg.ServiceReject != nil && !integrityRequiredCause(msg.ServiceReject.GetCauseValue())
	default:
		return true
	}
}

func integrityRequiredCause(cause uint8) bool {
	return cause == cause5GMMCAGNotAuthorized || cause == cause5GMMPlmnNotAllowedLoc
}
