// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

// GNodeB decodes through gNodeBConfig and copies field by field, so the yaml tags on GNodeB
// itself never run. A field added to GNodeB alone parses as absent and silently takes its zero
// value, which reads on a cluster as the feature simply not working — the gNB admitted a QoS flow
// it had been configured to refuse, and nothing anywhere said why.
//
// This asserts the two halves stay in step. It is worth having for any field added later, not
// just these.
func TestModificationOptionsSurviveTheConfigCopy(t *testing.T) {
	src := []byte(`
n2IpAddr: "1.2.3.4"
name: gnb1
modifyRejectQfis: [1, 5]
modifyRejectAll: true
`)

	var gnb GNodeB
	if err := yaml.Unmarshal(src, &gnb); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// A field that has always worked, so a failure below is about the new ones rather than the
	// decoding as a whole.
	if gnb.GnbName != "gnb1" {
		t.Fatalf("name = %q, want gnb1: the decode itself is broken, not just the new fields", gnb.GnbName)
	}

	if len(gnb.ModifyRejectQfis) != 2 || gnb.ModifyRejectQfis[0] != 1 || gnb.ModifyRejectQfis[1] != 5 {
		t.Errorf("modifyRejectQfis = %v, want [1 5]; check it is on gNodeBConfig and copied in UnmarshalYAML",
			gnb.ModifyRejectQfis)
	}
	if !gnb.ModifyRejectAll {
		t.Error("modifyRejectAll did not survive the copy; check gNodeBConfig and UnmarshalYAML")
	}
}
