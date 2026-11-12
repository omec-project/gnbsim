// Copyright 2024 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
package common

import (
	"fmt"
	"strings"
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

// The 5GMM events are numbered like the NAS message types of TS 24.501 table 9.7.1, because the
// event of a received NAS message is built as EventType(msgType)|N1_EVENT.
func TestFiveGmmEventsMatchNasMessageTypes(t *testing.T) {
	tests := []struct {
		event   EventType
		msgType EventType
	}{
		{FIVE_GMM_STATUS_EVENT, 0x64},
		{NOTIFICATION_EVENT, 0x65},
		{NOTIFICATION_RESPONSE_EVENT, 0x66},
		{UL_NAS_TRANSPORT_EVENT, 0x67},
		{DL_NAS_TRANSPORT_EVENT, 0x68},
	}
	for _, tc := range tests {
		if want := tc.msgType | N1_EVENT; tc.event != want {
			t.Errorf("%v: got %#x, want %#x", tc.event, uint32(tc.event), uint32(want))
		}
	}
}

// An event received from the network that this application does not know must not terminate it.
func TestString_UnknownEventAndProcedureDoNotTerminate(t *testing.T) {
	if out := (N1_EVENT | 0x7f).String(); !strings.HasPrefix(out, "UNKNOWN-EVENT") {
		t.Errorf("unexpected event string %q", out)
	}
	if out := ProcedureType(200).String(); !strings.HasPrefix(out, "UNKNOWN-PROCEDURE") {
		t.Errorf("unexpected procedure string %q", out)
	}
}
