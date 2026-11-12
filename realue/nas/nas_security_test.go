// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"strings"
	"testing"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/security"
)

func newSecuredUe() *realuectx.RealUe {
	ue := realuectx.NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		nil, nil, "", "", "", "", nil)
	for i := range ue.KnasInt {
		ue.KnasInt[i] = byte(i + 1)
	}
	return ue
}

// protectedDeregAccept returns a downlink "integrity protected" DEREGISTRATION ACCEPT with the
// given sequence number whose MAC is computed for count, or deliberately wrong if corrupt is set.
func protectedDeregAccept(t *testing.T, ue *realuectx.RealUe, sqn uint8, count uint32, corrupt bool) []byte {
	t.Helper()
	plain := []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}
	macInput := append([]byte{sqn}, plain...)
	mac, err := security.NASMacCalculate(ue.IntegrityAlg, ue.KnasInt, count, security.Bearer3GPP,
		security.DirectionDownlink, macInput)
	if err != nil {
		t.Fatalf("NASMacCalculate: %v", err)
	}
	if corrupt {
		mac[0] ^= 0xff
	}
	pdu := []byte{0x7e, nas.SecurityHeaderTypeIntegrityProtected}
	pdu = append(pdu, mac...)
	return append(pdu, macInput...)
}

// TS 24.501 4.4.4.2: a message that fails the integrity check is discarded, and the NAS COUNT
// is only updated after a successful check (4.4.3.3).
func TestNASDecodeDiscardsMessageWithBadMac(t *testing.T) {
	ue := newSecuredUe()
	ue.DLCount.Set(0, 5)

	pdu := protectedDeregAccept(t, ue, 6, 6, true)
	if _, err := NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu); err == nil {
		t.Fatal("expected an error for a message with a bad MAC")
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

// After the secure exchange has been established only the messages listed in TS 24.501
// 4.4.4.2 may arrive without integrity protection.
func TestNASDecodePlainNasAfterSecurityEstablished(t *testing.T) {
	ue := newSecuredUe()
	ue.NasSecurityEstablished = true

	allowed := []byte{0x7e, 0x00, nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, allowed); err != nil {
		t.Errorf("a plain DEREGISTRATION ACCEPT is allowed, got %v", err)
	}

	notAllowed := []byte{0x7e, 0x00, nas.MsgTypeConfigurationUpdateCommand}
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, notAllowed); err == nil {
		t.Error("a plain CONFIGURATION UPDATE COMMAND must be discarded once security is established")
	}

	ue.NasSecurityEstablished = false
	if _, err := NASDecode(ue, nas.SecurityHeaderTypePlainNas, notAllowed); err != nil && strings.Contains(err.Error(), "discarding") {
		t.Errorf("plain NAS must be accepted before security is established, got %v", err)
	}
}
