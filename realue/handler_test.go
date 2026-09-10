// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package realue

import (
	"testing"

	"github.com/omec-project/nas/v2/nasMessage"
)

// TestResolveModificationRequestType drives the decision the UE makes about the Request type IE,
// rather than the builder that receives it.
//
// The builder tests in realue/nas cover what each value produces, and they pass a value in
// directly -- so none of them can fail for a caller that never produces the value. The case that
// matters is the one those tests cannot see: a profile setting modificationRequestType and
// omitModificationRequestType together, which is how they read sitting side by side in the config.
func TestResolveModificationRequestType(t *testing.T) {
	const omitted = 0

	tests := []struct {
		name       string
		configured uint8
		omit       bool
		want       uint8
	}{
		{
			name: "unset means modification request",
			want: nasMessage.ULNASTransportRequestTypeModificationRequest,
		},
		{
			name:       "a configured value is kept",
			configured: nasMessage.ULNASTransportRequestTypeInitialRequest,
			want:       nasMessage.ULNASTransportRequestTypeInitialRequest,
		},
		{
			name: "omit with nothing configured leaves the IE out",
			omit: true,
			want: omitted,
		},
		{
			name:       "omit wins over a configured value",
			configured: nasMessage.ULNASTransportRequestTypeModificationRequest,
			omit:       true,
			want:       omitted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveModificationRequestType(tt.configured, tt.omit); got != tt.want {
				t.Errorf("resolveModificationRequestType(%d, %v) = %d, want %d",
					tt.configured, tt.omit, got, tt.want)
			}
		})
	}
}
