// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"
	"testing"

	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/ngap/v2/ngapType"
)

// TestQosFlowsConcurrentAccess covers the pairing a PDU session modification creates: the control
// plane records an admitted QoS flow while this session's user plane worker is reading the map for
// every uplink packet it forwards.
//
// Before the modification handling existed, every AddQosFlow ran at PDU session resource setup or
// handover request, both of which finish before the user plane worker goroutine is started. So the
// map had one writer and then only readers, and needed no guard. A network-requested modification
// arrives whenever the core decides to send one, which includes the middle of a data transfer.
//
// Strip the locking and this fails every run under -race, which is how the Makefile and CI run
// the suite. Without -race it is the same defect with a worse failure: an unguarded run threw a
// fatal "concurrent map read and map write" once in twenty, and a fatal is not recoverable.
func TestQosFlowsConcurrentAccess(t *testing.T) {
	ue := &GnbUpUe{
		QosFlows: make(map[int64]*ngapType.QosFlowSetupRequestItem),
		Log:      logger.GNodeBLog,
	}
	ue.AddQosFlow(1, &ngapType.QosFlowSetupRequestItem{})

	const iterations = 1000
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for i := range iterations {
			ue.AddQosFlow(int64(i%8), &ngapType.QosFlowSetupRequestItem{})
		}
	}()
	go func() {
		defer wg.Done()
		for range iterations {
			if qfi := ue.AnyQfi(); qfi < 0 {
				t.Errorf("AnyQfi returned %d", qfi)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := range iterations {
			ue.GetQosFlow(int64(i % 8))
		}
	}()
	go func() {
		defer wg.Done()
		for i := range iterations {
			ue.RemoveQosFlow(int64(i % 8))
		}
	}()

	wg.Wait()
}

// TestRemoveQosFlow covers what a modification carrying a QoS Flow to Release List has to leave
// behind: a flow the core has withdrawn must stop being offered to the uplink path, or the gNB
// goes on stamping a QFI the user plane has no rule for.
func TestRemoveQosFlow(t *testing.T) {
	ue := &GnbUpUe{
		QosFlows: make(map[int64]*ngapType.QosFlowSetupRequestItem),
		Log:      logger.GNodeBLog,
	}
	ue.AddQosFlow(2, &ngapType.QosFlowSetupRequestItem{})
	ue.AddQosFlow(3, &ngapType.QosFlowSetupRequestItem{})

	ue.RemoveQosFlow(2)

	if got := ue.GetQosFlow(2); got != nil {
		t.Error("GetQosFlow(2) still returns a flow the core released")
	}
	if got := ue.GetQosFlow(3); got == nil {
		t.Error("GetQosFlow(3) returns nothing: releasing one flow removed another")
	}
	if got := ue.AnyQfi(); got != 3 {
		t.Errorf("AnyQfi() = %d, want 3: the released flow must not be offered to the uplink", got)
	}
}

// TestAnyQfiWithoutFlows pins the answer for a session that has no recorded flow, which is what
// the uplink path saw when it ranged the map itself.
func TestAnyQfiWithoutFlows(t *testing.T) {
	ue := &GnbUpUe{
		QosFlows: make(map[int64]*ngapType.QosFlowSetupRequestItem),
		Log:      logger.GNodeBLog,
	}
	if qfi := ue.AnyQfi(); qfi != 0 {
		t.Errorf("AnyQfi() = %d, want 0 for a session with no flows", qfi)
	}
}
