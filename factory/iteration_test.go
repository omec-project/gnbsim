// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"testing"

	"github.com/omec-project/gnbsim/common"
)

func TestParseIterationStep(t *testing.T) {
	proc, wait, err := parseIterationStep("REGISTRATION-PROCEDURE 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc != common.REGISTRATION_PROCEDURE || wait != 5 {
		t.Errorf("got (%v, %d), want (%v, 5)", proc, wait, common.REGISTRATION_PROCEDURE)
	}

	// these used to panic (missing wait) or be accepted silently (unknown procedure)
	for _, step := range []string{
		"REGISTRATION-PROCEDURE",
		"REGISTRATION-PROCEDURE five",
		"REGISTRATION-PROCEDURE -1",
		"REGISTRATION-PROCEDURE 5 6",
		"NO-SUCH-PROCEDURE 5",
		"UNKNOWN-PROCEDURE 5",
	} {
		if _, _, err := parseIterationStep(step); err == nil {
			t.Errorf("expected an error for %q", step)
		}
	}
}
