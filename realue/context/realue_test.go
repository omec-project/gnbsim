// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"testing"

	"github.com/omec-project/nas/v2/security"
	"github.com/omec-project/openapi/v2/models"
)

func newKeyedUe() *RealUe {
	ue := NewRealUe("imsi-208930100007487", security.AlgCiphering128NEA2, security.AlgIntegrity128NIA2,
		nil, nil, "", "", "", "", nil)
	ue.Kamf = make([]byte, 32)
	for i := range ue.Kamf {
		ue.Kamf[i] = byte(i + 1)
	}
	return ue
}

// Derived keys wait for the SECURITY MODE COMMAND that takes them into use. Deriving the same keys
// again (a repeated AUTHENTICATION REQUEST) is not a new context and must leave the state alone,
// or a recorded SECURITY MODE COMMAND would verify again.
func TestDerivateAlgKeyLeavesContextInUseAlone(t *testing.T) {
	ue := newKeyedUe()
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending() || ue.NasKeysDerived {
		t.Fatal("the first keys must be pending, not in use")
	}
	pending, _, inUse, _ := ue.PendingNasKeys()
	if inUse {
		t.Error("keys that were never in use have no accepted COUNT")
	}

	// the SECURITY MODE COMMAND takes the keys into use
	ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})
	if ue.NewNasSecurityContextPending() || !ue.NasKeysDerived || !ue.DLCountInUse {
		t.Fatal("the keys must be in use and no longer pending")
	}
	if ue.KnasInt != pending {
		t.Fatal("the keys in use are not the ones that were pending")
	}

	ue.DerivateAlgKey()
	if ue.NewNasSecurityContextPending() || !ue.DLCountInUse {
		t.Error("deriving the same keys again must not re-arm the COUNT reset or clear the replay state")
	}

	// different keys are pending and do not replace the ones in use
	ue.Kamf[0] ^= 0xff
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending() {
		t.Error("different keys must be pending")
	}
	if ue.KnasInt != pending || !ue.DLCountInUse {
		t.Error("pending keys must not replace the context in use")
	}
}

// Keys staged by an earlier authentication are stale once a later one derives the keys in use.
func TestDerivateAlgKeyDropsStalePendingKeys(t *testing.T) {
	ue := newKeyedUe()
	ue.DerivateAlgKey()
	ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})

	kamf := append([]byte(nil), ue.Kamf...)
	ue.Kamf[0] ^= 0xff
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending() {
		t.Fatal("different keys must be pending")
	}

	ue.Kamf = kamf
	ue.DerivateAlgKey()
	if ue.NewNasSecurityContextPending() {
		t.Error("keys of a superseded authentication must not stay pending")
	}
}

// Moving to another security context and back puts the COUNT state of the first one back, so what
// was accepted in it stays accepted.
func TestActivatePendingNasContextRestoresStateOfEarlierContext(t *testing.T) {
	ue := newKeyedUe()
	ue.DerivateAlgKey()
	ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})
	ue.DLCount.Set(1, 7)
	ue.ULCount.Set(0, 9)
	keysA := ue.KnasInt

	ue.Kamf[0] ^= 0xff // context B
	ue.DerivateAlgKey()
	ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})
	if ue.KnasInt == keysA {
		t.Fatal("context B was not taken into use")
	}
	if got := ue.ULCount.Get(); got != 0 {
		t.Errorf("UL COUNT = %d, want a context that was never in use to start at 0", got)
	}
	ue.ULCount.Set(0, 3)

	ue.Kamf[0] ^= 0xff // back to context A
	ue.DerivateAlgKey()
	if ue.KnasInt == keysA {
		t.Fatal("pending keys must not replace the ones in use")
	}
	keys, dl, inUse, ok := ue.PendingNasKeys()
	if !ok || keys != keysA {
		t.Fatal("the keys of context A were not derived again")
	}
	if !inUse || dl.Get() != 1<<8|7 {
		t.Errorf("pending COUNT state = %#x, in use %v, want 0x107, in use", dl.Get(), inUse)
	}

	var verified security.Count
	verified.Set(1, 8)
	ue.ActivatePendingNasContext(verified, models.NgKsi{})
	if ue.KnasInt != keysA || ue.NewNasSecurityContextPending() {
		t.Error("context A must be in use and no longer pending")
	}
	if got := ue.DLCount.Get(); got != 1<<8|8 {
		t.Errorf("DL COUNT = %#x, want 0x108", got)
	}
	if got := ue.ULCount.Get(); got != 9 {
		t.Errorf("UL COUNT = %d, want 9", got)
	}
}

// AUTHENTICATION REQUEST is not integrity protected, so what its keys leave behind has to stay
// bounded: only a context that a SECURITY MODE COMMAND took into use is remembered.
func TestDerivateAlgKeyDoesNotAccumulateUnactivatedContexts(t *testing.T) {
	ue := newKeyedUe()
	ue.DerivateAlgKey()
	ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})
	for i := 0; i < 50; i++ {
		ue.Kamf[1] = byte(i)
		ue.DerivateAlgKey()
	}
	if n := len(ue.nasContexts); n != 0 {
		t.Errorf("%d contexts remembered for keys that were never taken into use, want 0", n)
	}
}

// The key set identifier in the unprotected AUTHENTICATION REQUEST is not authenticated, so it is
// the one in the SECURITY MODE COMMAND that takes effect, and only with the keys it takes into use:
// requests sent before it still name the context that protects them.
func TestNgKsiComesFromTheSecurityModeCommand(t *testing.T) {
	ue := newKeyedUe()
	ksiA, ksiB := models.NgKsi{Ksi: 1}, models.NgKsi{Ksi: 2}
	ue.DerivateAlgKey()
	if ue.NgKsi != (models.NgKsi{}) {
		t.Errorf("NgKsi = %+v before the keys are in use, want it unchanged", ue.NgKsi)
	}
	ue.ActivatePendingNasContext(security.Count{}, ksiA)
	if ue.NgKsi != ksiA {
		t.Fatalf("NgKsi = %+v, want %+v once the keys are in use", ue.NgKsi, ksiA)
	}

	ue.Kamf[0] ^= 0xff
	ue.DerivateAlgKey()
	if ue.NgKsi != ksiA {
		t.Errorf("NgKsi = %+v while new keys are pending, want %+v", ue.NgKsi, ksiA)
	}
	ue.ActivatePendingNasContext(security.Count{}, ksiB)
	if ue.NgKsi != ksiB {
		t.Fatalf("NgKsi = %+v, want %+v", ue.NgKsi, ksiB)
	}

	// a context that comes back takes the identifier its command carries
	ue.Kamf[0] ^= 0xff
	ue.DerivateAlgKey()
	ue.ActivatePendingNasContext(security.Count{}, ksiB)
	if ue.NgKsi != ksiB {
		t.Errorf("NgKsi = %+v after moving back, want %+v", ue.NgKsi, ksiB)
	}
}

// What a UE remembers of the contexts it has used is bounded, without ever blocking a fresh
// authentication. The oldest context is retired: its keys cannot be staged again, as they would
// come back with a COUNT of zero, and keys staged by an earlier authentication do not survive a
// refusal.
func TestRetainedNasContextsAreBoundedAndRetiredKeysStayRetired(t *testing.T) {
	ue := newKeyedUe()
	activate := func(i int) {
		ue.Kamf[2], ue.Kamf[3] = byte(i), byte(i>>8)
		ue.DerivateAlgKey()
		if !ue.NewNasSecurityContextPending() {
			t.Fatalf("fresh keys %d were refused", i)
		}
		ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})
	}
	activate(0)
	firstKeys := ue.KnasInt
	for i := 1; i < maxRetainedNasContexts+5; i++ {
		activate(i)
	}
	if n := len(ue.nasContexts); n != maxRetainedNasContexts {
		t.Errorf("%d contexts retained, want %d", n, maxRetainedNasContexts)
	}

	// a retained context can be staged
	ue.Kamf[2], ue.Kamf[3] = byte(maxRetainedNasContexts+2), 0
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending() {
		t.Fatal("the keys of a retained context must be staged")
	}
	// the first context was retired: it cannot come back, and the stale pending keys do not stay
	ue.Kamf[2], ue.Kamf[3] = 0, 0
	ue.DerivateAlgKey()
	if ue.NewNasSecurityContextPending() {
		t.Error("the keys of a retired context must not be staged, nor the stale pending ones kept")
	}
	if ue.KnasInt == firstKeys {
		t.Error("retired keys are in use")
	}
	if len(ue.nasRetired) == 0 {
		t.Error("nothing was retired")
	}
}

// Only a bounded number of fingerprints of retired contexts is kept.
func TestRetiredNasKeysAreBounded(t *testing.T) {
	ue := newKeyedUe()
	for i := 0; i < maxRetiredNasKeys+maxRetainedNasContexts+50; i++ {
		ue.Kamf[2], ue.Kamf[3] = byte(i), byte(i>>8)
		ue.DerivateAlgKey()
		ue.ActivatePendingNasContext(security.Count{}, models.NgKsi{})
	}
	if len(ue.nasRetired) != maxRetiredNasKeys || len(ue.nasRetiredOrder) != maxRetiredNasKeys {
		t.Errorf("%d fingerprints kept, want %d", len(ue.nasRetired), maxRetiredNasKeys)
	}
}
