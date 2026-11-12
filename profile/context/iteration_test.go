// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package context

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

// A profile received over REST carries only the iteration definitions, so PIterations must be
// built from them when the profile is initialised, not only when the configuration file is.
func TestBuildIterations(t *testing.T) {
	newProfile := func(start, next string) *Profile {
		return &Profile{
			Name:           "p",
			StartIteration: start,
			Iterations: []*Iterations{
				{Name: "one", First: "REGISTRATION-PROCEDURE 0", Next: next},
				{Name: "two", First: "AN-RELEASE-PROCEDURE 10"},
			},
		}
	}

	p := newProfile("one", "two")
	if err := p.BuildIterations(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := p.PIterations["one"].ProcMap[1]; got != common.REGISTRATION_PROCEDURE {
		t.Errorf("iteration one step 1 = %v", got)
	}
	if got := p.PIterations["two"].NextItr; got != quit {
		t.Errorf("iteration two next = %q, want %q", got, quit)
	}

	if err := newProfile("missing", "two").BuildIterations(); err == nil {
		t.Error("expected an error for an undefined startiteration")
	}
	if err := newProfile("one", "missing").BuildIterations(); err == nil {
		t.Error("expected an error for an undefined next iteration")
	}
	if err := (&Profile{}).BuildIterations(); err != nil {
		t.Errorf("a profile without iterations must be accepted: %v", err)
	}
}

// The ICMP destination must be an IPv4 address, since the packet builder cannot cope with
// anything else. An empty value selects the default.
func TestInitValidatesDefaultAs(t *testing.T) {
	for _, tc := range []struct {
		defaultAs string
		wantErr   bool
	}{
		{defaultAs: "", wantErr: false},
		{defaultAs: "192.168.250.1", wantErr: false},
		{defaultAs: "2001:db8::1", wantErr: true},
		{defaultAs: "not-an-ip", wantErr: true},
	} {
		p := &Profile{Name: "p", ProfileType: REGISTER, DefaultAs: tc.defaultAs}
		if err := p.Init(); (err != nil) != tc.wantErr {
			t.Errorf("defaultAs %q: error = %v, want error: %v", tc.defaultAs, err, tc.wantErr)
		}
	}
}
