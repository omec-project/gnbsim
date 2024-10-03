// Copyright 2024 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
package common

import (
	"fmt"
	"testing"
)

func TestString_ValidEventTypeReturnsEventTypeString(t *testing.T) {
	for event_type, event_type_str := range evtStrMap {
		t.Run(
			fmt.Sprintf("Testing [%v]", event_type),
			func(t *testing.T) {
				out := event_type.String()
				if out != event_type_str {
					t.Errorf("Invalid event string %v for %v", out, event_type)
				}
			},
		)
	}
}
