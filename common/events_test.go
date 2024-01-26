// Copyright 2024 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
package common

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/omec-project/gnbsim/logger"
	"github.com/stretchr/testify/assert"
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

func TestString_InvalidEventTypeExitsWithLogMsg(t *testing.T) {
	var INVALID_EVENT EventType = 0x0
	var logBuf bytes.Buffer
	assert := assert.New(t)
	patchedExit := func(int) { panic("Dummy exit function") }

	logger.AppLog.Logger.SetOutput(&logBuf)
	logger.AppLog.Logger.ExitFunc = patchedExit

	assert.Panics(func() { _ = INVALID_EVENT.String() })
	assert.Contains(logBuf.String(), "Invalid Event ID: 0x0")
}
