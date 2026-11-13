// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"errors"
	"strings"
	"testing"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/security"
)

func newSecuredUe() *realuectx.RealUe {
	ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		nil, nil, "", "", "", "", nil)
	for i := range ue.KnasInt {
		ue.KnasInt[i] = byte(i + 1)
	}
	ue.NasKeysDerived = true
	return ue
}

// protectedDeregAccept returns a downlink "integrity protected" DEREGISTRATION ACCEPT with the
// given sequence number whose MAC is computed for count, or deliberately wrong if corrupt is set.
func protectedDeregAccept(t *testing.T, ue *realuectx.RealUe, sqn uint8, count uint32, corrupt bool) []byte {
	t.Helper()
	return protectedDeregAcceptWithHeader(t, ue, nas.SecurityHeaderTypeIntegrityProtected, sqn, count, corrupt)
}

// protectedDeregAcceptWithHeader is protectedDeregAccept with the given security header type.
func protectedDeregAcceptWithHeader(t *testing.T, ue *realuectx.RealUe, securityHeaderType, sqn uint8,
	count uint32, corrupt bool,
) []byte {
	t.Helper()
	plain := []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}
	return protectedMessage(t, ue, securityHeaderType, plain, sqn, count, corrupt)
}

// protectedMessage wraps a plain 5GMM message into a downlink security protected message with the
// given security header type and sequence number. The MAC is computed for count, or deliberately
// wrong if corrupt is set.
func protectedMessage(t *testing.T, ue *realuectx.RealUe, securityHeaderType uint8, plain []byte, sqn uint8,
	count uint32, corrupt bool,
) []byte {
	t.Helper()
	return protectedMessageWithKey(t, ue, ue.KnasInt, securityHeaderType, plain, sqn, count, corrupt)
}

// protectedMessageWithKey is protectedMessage with the MAC computed under the given integrity key.
func protectedMessageWithKey(t *testing.T, ue *realuectx.RealUe, key [16]uint8, securityHeaderType uint8,
	plain []byte, sqn uint8, count uint32, corrupt bool,
) []byte {
	t.Helper()
	macInput := append([]byte{sqn}, plain...)
	mac, err := security.NASMacCalculate(ue.IntegrityAlg, key, count, security.Bearer3GPP,
		security.DirectionDownlink, macInput)
	if err != nil {
		t.Fatalf("NASMacCalculate: %v", err)
	}
	if len(mac) == 0 { // null integrity: the MAC field is there, all zero
		mac = make([]byte, 4)
	}
	if corrupt {
		mac[0] ^= 0xff
	}
	pdu := []byte{0x7e, securityHeaderType}
	pdu = append(pdu, mac...)
	return append(pdu, macInput...)
}

// derivePending makes the UE derive NAS keys from a KAMF made of seed, which leaves them pending
// until a SECURITY MODE COMMAND takes them into use, and returns their integrity key.
func derivePending(t *testing.T, ue *realuectx.RealUe, seed byte) [16]uint8 {
	t.Helper()
	ue.Kamf = make([]byte, 32)
	for i := range ue.Kamf {
		ue.Kamf[i] = seed + byte(i)
	}
	ue.DerivateAlgKey()
	key, _, _, ok := ue.PendingNasKeys()
	if !ok {
		t.Fatal("the derived keys are not pending")
	}
	return key
}

// securityModeCommandFor returns a downlink SECURITY MODE COMMAND protected with key.
func securityModeCommandFor(t *testing.T, ue *realuectx.RealUe, key [16]uint8, sqn uint8, count uint32) []byte {
	t.Helper()
	return protectedMessageWithKey(t, ue, key, nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext,
		securityModeCommand, sqn, count, false)
}

// TS 24.501 4.4.4.2: a message that fails the integrity check is discarded, and the NAS COUNT
// is only updated after a successful check (4.4.3.3).
func TestNASDecodeDiscardsMessageWithBadMac(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(0, 5)

	pdu := protectedDeregAccept(t, ue, 6, 6, true)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); !errors.Is(err, ErrNasDiscarded) {
		t.Fatalf("expected a discard error for a message with a bad MAC, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 5 {
		t.Errorf("DL COUNT changed to %d by a message that failed the integrity check", got)
	}
	if ue.NasSecurityEstablished {
		t.Error("a message that failed the integrity check must not establish NAS security")
	}
}

func TestNASDecodeAcceptsMessageWithGoodMacAndUpdatesCount(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(0, 5)

	pdu := protectedDeregAccept(t, ue, 6, 6, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ue.DLCount.Get(); got != 6 {
		t.Errorf("DL COUNT = %d, want 6", got)
	}
	if !ue.NasSecurityEstablished {
		t.Error("a successfully integrity checked message must establish NAS security")
	}
}

func TestNASDecodeRejectsTruncatedInput(t *testing.T) {
	ue := newSecuredUe()
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, []byte{}); err == nil {
		t.Error("expected an error for an empty message")
	}
	short := []byte{0x7e, nas.SecurityHeaderTypeIntegrityProtected, 1, 2, 3}
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(short), short); err == nil {
		t.Error("expected an error for a truncated security protected message")
	}
	one := []byte{0x7e}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, one); err == nil || !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected a too-short error, got %v", err)
	}
}

// Only the messages listed in TS 24.501 4.4.4.2 may arrive without integrity protection.
func TestNASDecodePlainNasWithoutIntegrityProtection(t *testing.T) {
	ue := newSecuredUe()
	ue.NasSecurityEstablished = true

	// a plain DEREGISTRATION ACCEPT is only allowed for a non switch-off deregistration request
	allowed := []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, allowed); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("an unsolicited plain DEREGISTRATION ACCEPT must be discarded, got %v", err)
	}
	ue.NonSwitchOffDeregPending = true
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, allowed); err != nil {
		t.Errorf("a plain DEREGISTRATION ACCEPT for a pending non switch-off request is allowed, got %v", err)
	}

	notAllowed := []byte{0x7e, 0x00, nas.MsgTypeConfigurationUpdateCommand}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, notAllowed); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a plain CONFIGURATION UPDATE COMMAND must be discarded once security is established, got %v", err)
	}

	// the list holds before the secure exchange is established too: the messages outside it need
	// integrity protection to be processed at all, and the listed ones are accepted either way
	ue.NasSecurityEstablished = false
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, notAllowed); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a plain CONFIGURATION UPDATE COMMAND must be discarded before security is established, got %v", err)
	}
	authRequest := []byte{0x7e, 0x00, nas.MsgTypeAuthenticationRequest}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, authRequest); errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a plain AUTHENTICATION REQUEST must not be discarded, got %v", err)
	}
}

// TS 24.501 4.4.4.2 a): a plain IDENTITY REQUEST is only accepted if SUCI is requested.
func TestNASDecodePlainIdentityRequestAfterSecurityEstablished(t *testing.T) {
	ue := newSecuredUe()
	ue.NasSecurityEstablished = true

	for _, tc := range []struct {
		name       string
		identity   uint8
		wantAccept bool
	}{
		{"SUCI", nasMessage.MobileIdentity5GSTypeSuci, true},
		{"5G-GUTI", nasMessage.MobileIdentity5GSType5gGuti, false},
		{"IMEI", nasMessage.MobileIdentity5GSTypeImei, false},
		{"IMEISV", nasMessage.MobileIdentity5GSTypeImeisv, false},
	} {
		pdu := []byte{0x7e, 0x00, nas.MsgTypeIdentityRequest, tc.identity}
		_, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, pdu)
		if tc.wantAccept && err != nil {
			t.Errorf("%s: a plain IDENTITY REQUEST for SUCI is allowed, got %v", tc.name, err)
		}
		if !tc.wantAccept && !errors.Is(err, ErrNasDiscarded) {
			t.Errorf("%s: a plain IDENTITY REQUEST must be discarded, got %v", tc.name, err)
		}
	}
}

// A message of a type that may not be processed without integrity protection is discarded before
// its body is decoded, so a malformed body cannot turn it into a decode error.
func TestNASDecodeDiscardsMalformedDisallowedPlainNas(t *testing.T) {
	ue := newSecuredUe()
	ue.NasSecurityEstablished = true

	// REGISTRATION ACCEPT without its mandatory IEs
	pdu := []byte{0x7e, 0x00, nas.MsgTypeRegistrationAccept}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, pdu); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a malformed disallowed plain message must be discarded, got %v", err)
	}
	// header only: no message type at all
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, []byte{0x7e, 0x00}); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a plain message without a message type must be discarded, got %v", err)
	}
	// a 5GSM message is never accepted without integrity protection
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, []byte{0x2e, 0x01, 0x01, nas.MsgTypePDUSessionEstablishmentAccept}); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a plain 5GSM message must be discarded, got %v", err)
	}
	// an allowed type whose body cannot be decoded is not trusted either
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, []byte{0x7e, 0x00, nas.MsgTypeRegistrationReject}); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("an undecodable plain REGISTRATION REJECT must be discarded, got %v", err)
	}
}

// A plain REGISTRATION REJECT / SERVICE REJECT with cause #76 or #78, or an IDENTITY REQUEST for
// anything but the SUCI, is discarded on its header, even when the rest of the body is malformed and
// whether or not the secure exchange of NAS messages is established.
func TestNASDecodeDiscardsMalformedPlainMessageThatNeedsIntegrityProtection(t *testing.T) {
	for _, established := range []bool{false, true} {
		ue := newSecuredUe()
		ue.NasSecurityEstablished = established
		for _, pdu := range [][]byte{
			{0x7e, 0x00, nas.MsgTypeRegistrationReject, 76, 0x5f}, // truncated optional IE
			{0x7e, 0x00, nas.MsgTypeRegistrationReject, 78, 0x5f},
			{0x7e, 0x00, nas.MsgTypeServiceReject, 76, 0x50},
			{0x7e, 0x00, nas.MsgTypeServiceReject, 78, 0x50},
			{0x7e, 0x00, nas.MsgTypeIdentityRequest, nasMessage.MobileIdentity5GSType5gGuti},
		} {
			if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, pdu); !errors.Is(err, ErrNasDiscarded) {
				t.Errorf("established=%v pdu=%x: must be discarded, got %v", established, pdu, err)
			}
		}
	}
}

// 5GMM causes #76 and #78 require integrity protection of REGISTRATION REJECT / SERVICE REJECT.
func TestNASDecodePlainRejectCauseAfterSecurityEstablished(t *testing.T) {
	ue := newSecuredUe()
	ue.NasSecurityEstablished = true

	for _, tc := range []struct {
		name       string
		msgType    uint8
		cause      uint8
		wantAccept bool
	}{
		{"registration reject #11", nas.MsgTypeRegistrationReject, 11, true},
		{"registration reject #76", nas.MsgTypeRegistrationReject, 76, false},
		{"registration reject #78", nas.MsgTypeRegistrationReject, 78, false},
		{"service reject #9", nas.MsgTypeServiceReject, 9, true},
		{"service reject #76", nas.MsgTypeServiceReject, 76, false},
		{"service reject #78", nas.MsgTypeServiceReject, 78, false},
	} {
		pdu := []byte{0x7e, 0x00, tc.msgType, tc.cause}
		_, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, pdu)
		if tc.wantAccept && err != nil {
			t.Errorf("%s: allowed without integrity protection, got %v", tc.name, err)
		}
		if !tc.wantAccept && !errors.Is(err, ErrNasDiscarded) {
			t.Errorf("%s: must be discarded, got %v", tc.name, err)
		}
	}
}

// With the null integrity algorithm there is no MAC to verify, but a security protected message
// still has the full 7 octet header, and a truncated one must be rejected rather than have part of
// the header taken for the plain message.
func TestNASDecodeNullIntegrity(t *testing.T) {
	ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA0,
		nil, nil, "", "", "", "", nil)

	// EPD, security header type, MAC, sequence number, plain DEREGISTRATION ACCEPT
	pdu := []byte{
		0x7e, nas.SecurityHeaderTypeIntegrityProtected, 0, 0, 0, 0, 0x01, 0x7e, 0x00,
		nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration,
	}
	msg, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := msg.GmmHeader.GetMessageType(); got != nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration {
		t.Errorf("message type = %#x, want DEREGISTRATION ACCEPT", got)
	}
	if got := ue.DLCount.Get(); got != 1 {
		t.Errorf("DL COUNT = %d, want 1", got)
	}
	if ue.NasSecurityEstablished {
		t.Error("nothing was integrity checked, so NAS security must not be established")
	}

	// the header is cut short, so what follows it is not a message
	short := []byte{
		0x7e, nas.SecurityHeaderTypeIntegrityProtected, 0x00, 0x7e, 0x00,
		nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration,
	}
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(short), short); err == nil || !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected a too-short error for a truncated header, got %v", err)
	}
}

// plain SECURITY MODE COMMAND, the only message that may come with the "new 5G NAS security
// context" security header type: selected NAS security algorithms (5G-EA2, 5G-IA2), ngKSI and the
// replayed UE security capabilities
var securityModeCommand = []byte{0x7e, 0x00, nas.MsgTypeSecurityModeCommand, 0x22, 0x00, 0x02, 0xf0, 0xf0}

// The NAS COUNT is rolled forward on a copy, so a message that fails the integrity check must
// leave the whole COUNT (overflow and sequence number) untouched, also when the sequence number
// wraps and when the header asks for the COUNT to be reset for a new security context. The same
// messages with a correct MAC must advance it as described.
func TestNASDecodeDLCountOverflowAndNewContext(t *testing.T) {
	count := func(overflow uint16, sqn uint8) uint32 { return uint32(overflow)<<8 | uint32(sqn) }

	for _, tc := range []struct {
		name         string
		plain        []byte
		macCount     uint32
		wantAfterOk  uint32
		wantAfterBad uint32
		startOvf     uint16
		header       uint8
		startSqn     uint8
		sqn          uint8
	}{
		{
			name:   "sequence number wraps",
			header: nas.SecurityHeaderTypeIntegrityProtected,
			plain:  []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration},
			// 0x02fa -> SQN 3 is lower than 0xfa, so the overflow counter advances
			startOvf: 2, startSqn: 0xfa, sqn: 3,
			macCount: count(3, 3), wantAfterOk: count(3, 3), wantAfterBad: count(2, 0xfa),
		},
		{
			name:     "new security context resets the overflow",
			header:   nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext,
			plain:    securityModeCommand,
			startOvf: 2, startSqn: 0xfa, sqn: 0,
			macCount: count(0, 0), wantAfterOk: count(0, 0), wantAfterBad: count(2, 0xfa),
		},
		{
			name:     "new security context with a non-zero sequence number",
			header:   nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext,
			plain:    securityModeCommand,
			startOvf: 2, startSqn: 0xfa, sqn: 7,
			macCount: count(0, 7), wantAfterOk: count(0, 7), wantAfterBad: count(2, 0xfa),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ue := newSecuredUe()
			ue.DLCount.Set(tc.startOvf, tc.startSqn)
			// new keys are only awaiting their SECURITY MODE COMMAND in the new context cases
			key := ue.KnasInt
			if tc.header == nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext {
				key = derivePending(t, ue, 1)
			}

			bad := protectedMessageWithKey(t, ue, key, tc.header, tc.plain, tc.sqn, tc.macCount, true)
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(bad), bad); !errors.Is(err, ErrNasDiscarded) {
				t.Fatalf("expected a discard error for a message with a bad MAC, got %v", err)
			}
			if got := ue.DLCount.Get(); got != tc.wantAfterBad {
				t.Errorf("DL COUNT = %#x after a discarded message, want %#x", got, tc.wantAfterBad)
			}

			good := protectedMessageWithKey(t, ue, key, tc.header, tc.plain, tc.sqn, tc.macCount, false)
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(good), good); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := ue.DLCount.Get(); got != tc.wantAfterOk {
				t.Errorf("DL COUNT = %#x after an accepted message, want %#x", got, tc.wantAfterOk)
			}
		})
	}
}

// The outer security header type is not covered by the MAC. Changing only that octet of a recorded
// message must therefore not reset the COUNT, or the recorded message would verify again at the
// COUNT it was originally sent with.
func TestNASDecodeHeaderTypeChangeDoesNotResetCount(t *testing.T) {
	ue := newSecuredUe()
	derivePending(t, ue, 1)
	ue.DLCount.Set(2, 0xfa) // live COUNT 0x02fa

	// a genuine message that was sent at COUNT 7 under the same keys, then recorded
	recorded := protectedDeregAccept(t, ue, 7, 7, false)
	for _, tc := range []struct {
		name   string
		header uint8
	}{
		{"new context", nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext},
		{"ciphered with new context", nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext},
	} {
		replayed := append([]byte(nil), recorded...)
		replayed[1] = tc.header
		if _, err := NASDecode(ue, nas.GetSecurityHeaderType(replayed), replayed); !errors.Is(err, ErrNasDiscarded) {
			t.Errorf("%s: expected the replayed message to be discarded, got %v", tc.name, err)
		}
		if got := ue.DLCount.Get(); got != 0x02fa {
			t.Errorf("%s: DL COUNT = %#x, want it unchanged at 0x2fa", tc.name, got)
		}
	}
}

// A SECURITY MODE COMMAND with the new context header type takes the pending keys into use and
// resets the COUNT, once. A retransmission (TS 24.501 5.4.2.4) has a COUNT of its own and is
// accepted under the keys now in use without a reset; a replay of the first one is not.
func TestNASDecodeSecurityModeCommandAndRetransmission(t *testing.T) {
	const newContext = nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext

	ue := newSecuredUe()
	ue.DLCount.Set(2, 0xfa) // the COUNT of the previous context
	smc := securityModeCommandFor(t, ue, derivePending(t, ue, 1), 0, 0)

	msg, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.SecurityModeCommand == nil {
		t.Error("the SECURITY MODE COMMAND was not decoded")
	}
	if got := ue.DLCount.Get(); got != 0 {
		t.Errorf("DL COUNT = %#x, want it reset to 0", got)
	}
	if ue.NewNasSecurityContextPending() {
		t.Error("the new security context must no longer be pending once it is in use")
	}

	// SECURITY MODE COMPLETE got lost: the network retransmits the command with the next COUNT
	retransmitted := protectedMessage(t, ue, newContext, securityModeCommand, 1, 1, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(retransmitted), retransmitted); err != nil {
		t.Fatalf("a retransmitted SECURITY MODE COMMAND must be accepted, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 1 {
		t.Errorf("DL COUNT = %#x after the retransmission, want 1 (no reset)", got)
	}

	// the first command, replayed, no longer verifies: it estimates to a higher COUNT than it was sent at
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a replayed SECURITY MODE COMMAND must be discarded, got %v", err)
	}
	// and so does the retransmission itself
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(retransmitted), retransmitted); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a replayed retransmission must be discarded, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 1 {
		t.Errorf("DL COUNT = %#x after the replays, want 1", got)
	}
}

// Keys staged by an unprotected AUTHENTICATION REQUEST must not hide the context in use: a valid
// retransmission of its SECURITY MODE COMMAND is still accepted, and the pending keys stay pending.
func TestNASDecodeSecurityModeCommandRetransmissionWithOtherKeysPending(t *testing.T) {
	const newContext = nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext

	ue := newSecuredUe()
	smc := securityModeCommandFor(t, ue, derivePending(t, ue, 1), 0, 0)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inUse := ue.KnasInt

	// another authentication stages different keys
	pending := derivePending(t, ue, 100)
	if pending == inUse {
		t.Fatal("test needs different keys")
	}

	retransmitted := protectedMessage(t, ue, newContext, securityModeCommand, 1, 1, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(retransmitted), retransmitted); err != nil {
		t.Fatalf("a retransmitted SECURITY MODE COMMAND must be accepted, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 1 || ue.KnasInt != inUse {
		t.Errorf("DL COUNT = %#x, want 1 with the keys in use unchanged", got)
	}
	if !ue.NewNasSecurityContextPending() {
		t.Error("the pending keys must stay pending")
	}

	// a replay of it is still discarded, and the command for the pending keys still activates them
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(retransmitted), retransmitted); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a replayed retransmission must be discarded, got %v", err)
	}
	next := securityModeCommandFor(t, ue, pending, 0, 0)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(next), next); err != nil {
		t.Fatalf("the command for the pending keys must be accepted, got %v", err)
	}
	if ue.KnasInt != pending || ue.NewNasSecurityContextPending() {
		t.Error("the pending keys must be in use after their command")
	}
}

// Without new keys a SECURITY MODE COMMAND does not reset the COUNT: one that was not sent at the
// COUNT the UE estimates fails the integrity check.
func TestNASDecodeSecurityModeCommandWithoutNewKeysDoesNotResetCount(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(2, 0xfa)
	ue.DLCountInUse = true
	smc := protectedMessage(t, ue, nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext,
		securityModeCommand, 0, 0, false)

	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); !errors.Is(err, ErrNasDiscarded) {
		t.Fatalf("expected the SECURITY MODE COMMAND to be discarded, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 0x02fa {
		t.Errorf("DL COUNT = %#x after a discarded message, want 0x2fa", got)
	}
}

// TS 24.501 4.4.3.2: a given NAS COUNT is accepted at most once. An exact replay of the last
// accepted message estimates to the same COUNT, with the same MAC, so it has to be refused on that
// ground alone.
func TestNASDecodeDiscardsReplayOfAcceptedCount(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(0, 5)
	ue.DLCountInUse = true

	pdu := protectedDeregAccept(t, ue, 6, 6, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); !errors.Is(err, ErrNasDiscarded) {
		t.Fatalf("a replayed message must be discarded, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 6 {
		t.Errorf("DL COUNT = %d after a replay, want 6", got)
	}

	// the next message of the sender is still accepted
	next := protectedDeregAccept(t, ue, 7, 7, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(next), next); err != nil {
		t.Errorf("the next message must be accepted, got %v", err)
	}
}

// COUNT 0 is a valid COUNT for the first message of a security context, which has to be told apart
// from a COUNT that was already accepted: it is accepted once, and only once.
func TestNASDecodeInitialCountZeroIsAcceptedOnce(t *testing.T) {
	ue := newSecuredUe()
	pdu := protectedDeregAccept(t, ue, 0, 0, false)

	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
		t.Fatalf("the first message at COUNT 0 must be accepted, got %v", err)
	}
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a replay of COUNT 0 must be discarded, got %v", err)
	}
}

// Replay protection does not apply with the null integrity algorithm (TS 24.501 4.4.3.2).
func TestNASDecodeNullIntegrityHasNoReplayProtection(t *testing.T) {
	ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA0,
		nil, nil, "", "", "", "", nil)
	pdu := []byte{
		0x7e, nas.SecurityHeaderTypeIntegrityProtected, 0, 0, 0, 0, 0x01, 0x7e, 0x00,
		nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration,
	}
	for i := 0; i < 2; i++ {
		if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i+1, err)
		}
	}
}

// The security header type is not covered by the MAC. With fresh keys and nothing accepted yet, a
// SECURITY MODE COMMAND whose type is altered must not be taken for one that takes the keys into
// use: it is checked under the keys in use, where it fails, and leaves the pending keys alone.
func TestNASDecodeSecurityModeCommandWithAlteredHeaderType(t *testing.T) {
	const newContext = nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext

	for _, tc := range []struct {
		name   string
		header uint8
	}{
		{"integrity protected", nas.SecurityHeaderTypeIntegrityProtected},
		{"integrity protected and ciphered", nas.SecurityHeaderTypeIntegrityProtectedAndCiphered},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ue := newSecuredUe()
			original := securityModeCommandFor(t, ue, derivePending(t, ue, 1), 0, 0)

			altered := append([]byte(nil), original...)
			altered[1] = tc.header
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(altered), altered); !errors.Is(err, ErrNasDiscarded) {
				t.Fatalf("the altered command must be discarded, got %v", err)
			}
			if !ue.NewNasSecurityContextPending() || ue.DLCountInUse {
				t.Error("a discarded message must leave the pending state alone")
			}

			// the genuine command is accepted once ...
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(original), original); err != nil {
				t.Fatalf("the genuine command must be accepted, got %v", err)
			}
			// ... and not again, whatever its header type claims
			for _, header := range []uint8{newContext, tc.header} {
				replayed := append([]byte(nil), original...)
				replayed[1] = header
				if _, err := NASDecode(ue, nas.GetSecurityHeaderType(replayed), replayed); !errors.Is(err, ErrNasDiscarded) {
					t.Errorf("a replay with header type %d must be discarded, got %v", header, err)
				}
			}
		})
	}
}

// The uplink COUNT starts over once, when the new security context is taken into use. The
// response to a retransmitted SECURITY MODE COMMAND must go out with the next COUNT, not reuse
// COUNT 0 under the same keys, and answering must leave the downlink COUNT alone, or the command
// could be replayed afterwards.
func TestNASEncodeSecurityModeCompleteUsesNextUplinkCount(t *testing.T) {
	const newContext = nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext

	ue := newSecuredUe()
	ue.ULCount.Set(0, 9) // the uplink COUNT of the previous context
	smc := securityModeCommandFor(t, ue, derivePending(t, ue, 1), 0, 0)
	if got := ue.ULCount.Get(); got != 9 {
		t.Fatalf("UL COUNT = %d before the command is accepted, want the 9 of the context in use", got)
	}
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ue.ULCount.Get(); got != 0 {
		t.Fatalf("UL COUNT = %d after the new context is taken into use, want it reset to 0", got)
	}

	complete := []byte{0x7e, 0x00, nas.MsgTypeSecurityModeComplete}
	respond := func(wantSqn uint8) {
		t.Helper()
		pdu, err := EncodeNasPduWithSecurity(ue, complete,
			nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true)
		if err != nil || len(pdu) < minSecurityProtectedLen {
			t.Fatalf("EncodeNasPduWithSecurity() = %x, %v", pdu, err)
		}
		if pdu[6] != wantSqn {
			t.Errorf("sequence number of the SECURITY MODE COMPLETE = %d, want %d", pdu[6], wantSqn)
		}
	}
	respond(0)

	retransmitted := protectedMessage(t, ue, newContext, securityModeCommand, 1, 1, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(retransmitted), retransmitted); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	respond(1)
	if got := ue.ULCount.Get(); got != 2 {
		t.Errorf("UL COUNT = %d after two responses, want 2", got)
	}
	if got := ue.DLCount.Get(); got != 1 {
		t.Errorf("DL COUNT = %d after answering, want it left at 1", got)
	}

	for name, replay := range map[string][]byte{"original command": smc, "retransmission": retransmitted} {
		if _, err := NASDecode(ue, nas.GetSecurityHeaderType(replay), replay); !errors.Is(err, ErrNasDiscarded) {
			t.Errorf("a replayed %s must be discarded after the SECURITY MODE COMPLETE, got %v", name, err)
		}
	}
}

// Before NAS keys have been derived the integrity key is all zero, which anyone can use to compute
// a valid MAC. A protected message must not be accepted on that basis, nor establish NAS security.
// Derived keys are only pending: until a SECURITY MODE COMMAND takes them into use they do not
// verify ordinary messages either.
func TestNASDecodeDiscardsProtectedMessageBeforeKeysAreDerived(t *testing.T) {
	ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		nil, nil, "", "", "", "", nil)
	pdu := protectedDeregAccept(t, ue, 0, 0, false) // a valid MAC, under the zero key

	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); !errors.Is(err, ErrNasDiscarded) {
		t.Fatalf("a protected message before the keys are derived must be discarded, got %v", err)
	}
	if ue.NasSecurityEstablished || ue.DLCountInUse {
		t.Error("a discarded message must not establish NAS security or use up a COUNT")
	}

	pendingKey := derivePending(t, ue, 1)
	ordinary := protectedMessageWithKey(t, ue, pendingKey, nas.SecurityHeaderTypeIntegrityProtected,
		[]byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}, 0, 0, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(ordinary), ordinary); !errors.Is(err, ErrNasDiscarded) {
		t.Fatalf("a message under keys that are not in use yet must be discarded, got %v", err)
	}

	smc := securityModeCommandFor(t, ue, pendingKey, 0, 0)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); err != nil {
		t.Fatalf("the SECURITY MODE COMMAND must be accepted under the pending keys, got %v", err)
	}
	if !ue.NasSecurityEstablished {
		t.Error("the secure exchange of NAS messages must be established by the SECURITY MODE COMMAND")
	}
	next := protectedDeregAccept(t, ue, 1, 1, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(next), next); err != nil {
		t.Errorf("unexpected error once the keys are in use: %v", err)
	}
}

// The SECURITY MODE COMMAND must pass the integrity check before anything is done with it, so a
// plain one is never processed (it is not among the messages of TS 24.501 4.4.4.2 that are
// accepted unprotected). That holds before and after the secure exchange has been established,
// and with new keys pending, where the SECURITY MODE COMPLETE would otherwise be sent for it.
func TestNASDecodeDiscardsPlainSecurityModeCommand(t *testing.T) {
	for _, tc := range []struct {
		setup func(t *testing.T, ue *realuectx.RealUe)
		name  string
	}{
		{name: "nothing derived", setup: func(*testing.T, *realuectx.RealUe) {}},
		{name: "keys pending", setup: func(t *testing.T, ue *realuectx.RealUe) { derivePending(t, ue, 1) }},
		{name: "secure exchange established", setup: func(_ *testing.T, ue *realuectx.RealUe) {
			ue.NasSecurityEstablished = true
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ue := newSecuredUe()
			tc.setup(t, ue)
			_, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, securityModeCommand)
			if !errors.Is(err, ErrNasDiscarded) {
				t.Fatalf("a plain SECURITY MODE COMMAND must be discarded, got %v", err)
			}
			if ue.NasSecurityEstablished != (tc.name == "secure exchange established") {
				t.Error("a discarded message must not change whether NAS security is established")
			}
		})
	}
}

// Deriving keys for a reauthentication must not disturb the context in use: ordinary messages,
// including a retransmitted AUTHENTICATION REQUEST, keep verifying under its keys and COUNT until
// a SECURITY MODE COMMAND takes the new keys into use.
func TestNASDecodeKeepsVerifyingWithKeysInUseWhileNewKeysArePending(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(0, 5)
	ue.DLCountInUse = true
	ue.ULCount.Set(0, 7)
	keys := ue.KnasInt
	pendingKey := derivePending(t, ue, 1)

	if ue.KnasInt != keys || ue.DLCount.Get() != 5 || ue.ULCount.Get() != 7 {
		t.Fatal("deriving keys changed the context in use")
	}
	pdu := protectedDeregAccept(t, ue, 6, 6, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
		t.Fatalf("a message under the keys in use must be accepted while keys are pending, got %v", err)
	}
	if !ue.NewNasSecurityContextPending() {
		t.Error("an ordinary message must not consume the pending keys")
	}

	smc := securityModeCommandFor(t, ue, pendingKey, 0, 0)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ue.KnasInt != pendingKey {
		t.Error("the SECURITY MODE COMMAND must take the pending keys into use")
	}
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); !errors.Is(err, ErrNasDiscarded) {
		t.Errorf("a message under the old keys must be discarded once the new ones are in use, got %v", err)
	}
}

// A -> B -> A: after moving to another security context, a replayed AUTHENTICATION REQUEST for the
// first one derives its keys again. The context's COUNT state must come back with them when a
// SECURITY MODE COMMAND activates it, or its recorded messages (SECURITY MODE COMMAND included)
// would verify again at their original COUNT.
func TestNASDecodeRecordedMessagesStayRejectedAfterContextRollback(t *testing.T) {
	ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		nil, nil, "", "", "", "", nil)
	accept := func(pdu []byte) {
		t.Helper()
		if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// context A: its command and a message after it are recorded
	keyA := derivePending(t, ue, 1)
	smcA := securityModeCommandFor(t, ue, keyA, 0, 0)
	accept(smcA)
	msgA := protectedDeregAccept(t, ue, 1, 1, false)
	accept(msgA)
	ue.ULCount.Set(0, 5)

	// context B
	accept(securityModeCommandFor(t, ue, derivePending(t, ue, 100), 0, 0))
	ue.ULCount.Set(0, 2)

	// the AUTHENTICATION REQUEST of A is replayed: A is pending, B stays in use
	if derivePending(t, ue, 1) != keyA {
		t.Fatal("the keys of context A were not derived again")
	}
	if ue.KnasInt == keyA {
		t.Fatal("deriving the keys must not put them into use")
	}
	for name, replay := range map[string][]byte{"command": smcA, "message": msgA} {
		if _, err := NASDecode(ue, nas.GetSecurityHeaderType(replay), replay); !errors.Is(err, ErrNasDiscarded) {
			t.Errorf("the recorded %s of context A must stay discarded, got %v", name, err)
		}
	}

	// the network takes A into use again with a command at the next COUNT of A
	accept(securityModeCommandFor(t, ue, keyA, 2, 2))
	if ue.KnasInt != keyA {
		t.Error("the SECURITY MODE COMMAND must take context A into use")
	}
	if got := ue.DLCount.Get(); got != 2 {
		t.Errorf("DL COUNT = %d, want 2", got)
	}
	if got := ue.ULCount.Get(); got != 5 {
		t.Errorf("UL COUNT = %d, want the 5 context A had reached, so no COUNT is used twice", got)
	}
	for name, replay := range map[string][]byte{"command": smcA, "message": msgA} {
		if _, err := NASDecode(ue, nas.GetSecurityHeaderType(replay), replay); !errors.Is(err, ErrNasDiscarded) {
			t.Errorf("the recorded %s of context A must stay discarded after it is back in use, got %v", name, err)
		}
	}
}

// When a context is restored as the pending one, its command is checked against the COUNT last
// accepted in that context, not the one of the context in use. A last accepted its command at
// COUNT 0 while B in use is at COUNT 1: A's recorded command must be discarded as a replay, and a
// fresh command of A at COUNT 1 must be accepted.
func TestNASDecodeRestoredContextIsCheckedAgainstItsOwnCount(t *testing.T) {
	for _, tc := range []struct {
		name    string
		count   uint32
		sqn     uint8
		accepts bool
	}{
		{name: "recorded command", sqn: 0, count: 0, accepts: false},
		{name: "fresh command", sqn: 1, count: 1, accepts: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
				nil, nil, "", "", "", "", nil)
			keyA := derivePending(t, ue, 1)
			smcA := securityModeCommandFor(t, ue, keyA, 0, 0)
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smcA), smcA); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			smcB := securityModeCommandFor(t, ue, derivePending(t, ue, 100), 0, 0)
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smcB), smcB); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			next := protectedDeregAccept(t, ue, 1, 1, false) // B is at COUNT 1
			if _, err := NASDecode(ue, nas.GetSecurityHeaderType(next), next); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			derivePending(t, ue, 1) // A is pending again
			pdu := securityModeCommandFor(t, ue, keyA, tc.sqn, tc.count)
			_, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
			if tc.accepts && err != nil {
				t.Errorf("a fresh command of the restored context must be accepted, got %v", err)
			}
			if !tc.accepts && !errors.Is(err, ErrNasDiscarded) {
				t.Errorf("a recorded command of the restored context must be discarded, got %v", err)
			}
		})
	}
}

// The NAS COUNT must not wrap around under the same keys (TS 24.501 clause 4.4.3.5): at the end of
// the COUNT range a sequence number of 0 would estimate to COUNT 0 again, and a message recorded
// when it was sent at COUNT 0 would verify a second time. Null integrity allows the wrap around.
func TestNASDecodeDiscardsDLCountWrapAround(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(0xffff, 0xff)
	ue.DLCountInUse = true
	recorded := protectedDeregAccept(t, ue, 0, 0, false) // genuinely protected at COUNT 0 long ago

	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(recorded), recorded); !errors.Is(err, ErrNasDiscarded) {
		t.Fatalf("a message that wraps the DL COUNT must be discarded, got %v", err)
	}
	if got := ue.DLCount.Get(); got != 0xffffff {
		t.Errorf("DL COUNT = %#x after a discarded message, want 0xffffff", got)
	}

	nullUe := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA0,
		nil, nil, "", "", "", "", nil)
	nullUe.DLCount.Set(0xffff, 0xff)
	pdu := []byte{
		0x7e, nas.SecurityHeaderTypeIntegrityProtected, 0, 0, 0, 0, 0x00, 0x7e, 0x00,
		nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration,
	}
	if _, err := NASDecode(nullUe, nas.GetSecurityHeaderType(pdu), pdu); err != nil {
		t.Fatalf("null integrity allows the COUNT to wrap around, got %v", err)
	}
	if got := nullUe.DLCount.Get(); got != 0 {
		t.Errorf("DL COUNT = %#x, want it wrapped to 0", got)
	}
}

// The uplink COUNT must not wrap around either: the message is not sent.
func TestNASEncodeRefusesToWrapUplinkCount(t *testing.T) {
	ue := newSecuredUe()
	ue.ULCount.Set(0xffff, 0xfe)
	plain := []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}
	if _, err := EncodeNasPduWithSecurity(ue, plain, nas.SecurityHeaderTypeIntegrityProtected, true); err != nil {
		t.Fatalf("COUNT 0xfffffe must still be usable, got %v", err)
	}
	if got := ue.ULCount.Get(); got != 0xffffff {
		t.Fatalf("UL COUNT = %#x, want 0xffffff", got)
	}
	if _, err := EncodeNasPduWithSecurity(ue, plain, nas.SecurityHeaderTypeIntegrityProtected, true); err == nil {
		t.Error("a message that would wrap the UL COUNT must not be sent")
	}
	if got := ue.ULCount.Get(); got != 0xffffff {
		t.Errorf("UL COUNT = %#x after the refusal, want it left alone", got)
	}
}

// The key set identifier of the new context is the one in the authenticated SECURITY MODE COMMAND.
func TestNASDecodeSecurityModeCommandSetsNgKsi(t *testing.T) {
	ue := newSecuredUe()
	command := append([]byte(nil), securityModeCommand...)
	command[4] = 0x03 // ngKSI 3, native
	smc := protectedMessageWithKey(t, ue, derivePending(t, ue, 1),
		nas.SecurityHeaderTypeIntegrityProtectedWithNew5gNasSecurityContext, command, 0, 0, false)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(smc), smc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ue.NgKsi.Ksi != 3 {
		t.Errorf("NgKsi = %+v, want ksi 3 from the SECURITY MODE COMMAND", ue.NgKsi)
	}
}

// protectedAndCipheredMessage is a downlink message ciphered with the UE's ciphering algorithm and
// key at count, with the MAC computed over the ciphertext.
func protectedAndCipheredMessage(t *testing.T, ue *realuectx.RealUe, plain []byte, sqn uint8, count uint32) []byte {
	t.Helper()
	ciphered := append([]byte(nil), plain...)
	if err := security.NASEncrypt(ue.CipheringAlg, ue.KnasEnc, count, security.Bearer3GPP,
		security.DirectionDownlink, ciphered); err != nil {
		t.Fatalf("NASEncrypt: %v", err)
	}
	if string(ciphered) == string(plain) {
		t.Fatal("the fixture is not actually ciphered")
	}
	return protectedMessage(t, ue, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, ciphered, sqn, count, false)
}

// Downlink messages that are really ciphered (5G-EA2) are decrypted with the accepted COUNT and the
// ciphering key, whether integrity is checked (5G-IA2) or null (5G-IA0), and only when the header
// says they are ciphered.
func TestNASDecodeDecryptsCipheredMessages(t *testing.T) {
	plain := []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}
	isAccept := func(msg *nas.Message) bool {
		return msg != nil && msg.GmmHeader.GetMessageType() == nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration
	}
	newUe := func(integrityAlg uint8) *realuectx.RealUe {
		ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA2, integrityAlg,
			nil, nil, "", "", "", "", nil)
		for i := range ue.KnasInt {
			ue.KnasInt[i] = byte(i + 1)
			ue.KnasEnc[i] = byte(0xa0 + i)
		}
		ue.NasKeysDerived = true
		ue.DLCount.Set(1, 0x10)
		ue.DLCountInUse = true
		return ue
	}

	for _, tc := range []struct {
		name         string
		integrityAlg uint8
	}{
		{"NEA2 with NIA2", security.AlgIntegrity128NIA2},
		{"NEA2 with NIA0", security.AlgIntegrity128NIA0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ue := newUe(tc.integrityAlg)
			pdu := protectedAndCipheredMessage(t, ue, plain, 0x11, 0x111)
			msg, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
			if err != nil || !isAccept(msg) {
				t.Fatalf("a ciphered message must be decrypted, got %v, %+v", err, msg)
			}
			if got := ue.DLCount.Get(); got != 0x111 {
				t.Errorf("DL COUNT = %#x, want 0x111", got)
			}

			// ciphered for another COUNT, with a MAC that is right for this one (or none under NIA0):
			// the keystream is wrong, so the message does not come out as it went in
			ue = newUe(tc.integrityAlg)
			ciphered := append([]byte(nil), plain...)
			if err := security.NASEncrypt(ue.CipheringAlg, ue.KnasEnc, 0x110, security.Bearer3GPP,
				security.DirectionDownlink, ciphered); err != nil {
				t.Fatal(err)
			}
			wrong := protectedMessage(t, ue, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, ciphered, 0x11, 0x111, false)
			if msg, err := NASDecode(ue, nas.GetSecurityHeaderType(wrong), wrong); err == nil && isAccept(msg) {
				t.Error("a message ciphered with another COUNT must not decrypt to the original")
			}
		})
	}

	// integrity protected only: the payload is plain even though a ciphering algorithm is configured
	t.Run("NIA0 without ciphering", func(t *testing.T) {
		ue := newUe(security.AlgIntegrity128NIA0)
		pdu := protectedMessage(t, ue, nas.SecurityHeaderTypeIntegrityProtected, plain, 0x11, 0x111, false)
		msg, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
		if err != nil || !isAccept(msg) {
			t.Fatalf("an unciphered message must not be decrypted, got %v, %+v", err, msg)
		}
	})
}
