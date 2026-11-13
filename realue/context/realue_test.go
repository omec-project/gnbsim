// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"testing"

	"github.com/omec-project/nas/v2/security"
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

// New keys start a new security context: the COUNT reset is armed and the accepted COUNTs are
// forgotten. Deriving the same keys again (a repeated AUTHENTICATION REQUEST) is not a new context
// and must leave that state alone, or a recorded SECURITY MODE COMMAND would verify again.
func TestDerivateAlgKeyOnlyStartsNewContextForNewKeys(t *testing.T) {
	ue := newKeyedUe()
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending || ue.DLCountInUse {
		t.Fatal("the first keys must start a new security context")
	}

	// the SECURITY MODE COMMAND takes the keys into use
	ue.NewNasSecurityContextPending = false
	ue.DLCountInUse = true
	keys := ue.KnasInt

	ue.DerivateAlgKey()
	if ue.NewNasSecurityContextPending || !ue.DLCountInUse {
		t.Error("deriving the same keys again must not re-arm the COUNT reset or clear the replay state")
	}
	if ue.KnasInt != keys {
		t.Error("the keys changed although the KAMF did not")
	}

	ue.Kamf[0] ^= 0xff
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending || ue.DLCountInUse {
		t.Error("different keys must start a new security context")
	}
	if ue.KnasInt == keys {
		t.Error("the keys did not change with the KAMF")
	}
}

// Moving to another security context and back puts the COUNT state of the first one back, so what
// was accepted in it stays accepted.
func TestDerivateAlgKeyRestoresStateOfEarlierContext(t *testing.T) {
	ue := newKeyedUe()
	ue.DerivateAlgKey()
	ue.NewNasSecurityContextPending = false
	ue.DLCountInUse = true
	ue.DLCount.Set(1, 7)
	ue.ULCount.Set(0, 9)
	keysA := ue.KnasInt

	ue.Kamf[0] ^= 0xff // context B
	ue.DerivateAlgKey()
	if !ue.NewNasSecurityContextPending || ue.DLCountInUse {
		t.Fatal("different keys must start a new security context")
	}
	ue.NewNasSecurityContextPending = false
	ue.DLCount.Set(0, 0)
	ue.ULCount.Set(0, 3)

	ue.Kamf[0] ^= 0xff // back to context A
	ue.DerivateAlgKey()
	if ue.KnasInt != keysA {
		t.Fatal("the keys of context A were not derived again")
	}
	if ue.NewNasSecurityContextPending || !ue.DLCountInUse {
		t.Error("context A must not start over")
	}
	if got := ue.DLCount.Get(); got != 1<<8|7 {
		t.Errorf("DL COUNT = %#x, want 0x107", got)
	}
	if got := ue.ULCount.Get(); got != 9 {
		t.Errorf("UL COUNT = %d, want 9", got)
	}
}
