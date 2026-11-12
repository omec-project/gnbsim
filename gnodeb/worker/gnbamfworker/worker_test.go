// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package gnbamfworker

import (
	"testing"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
)

// Once a context has been asked to quit, nothing new is queued for it, so traffic from the AMF
// cannot keep its queue from running empty, and a full queue does not hold up the receive loop.
func TestSendToGnbUeStopsQueueingOnceQuitIsRequested(t *testing.T) {
	gnbue := gnbctx.NewGnbCpUe(1, &gnbctx.GNodeB{}, nil)

	SendToGnbUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nil, 1)
	if len(gnbue.ReadChan) != 1 {
		t.Fatalf("a message for a live context was not queued")
	}

	gnbue.RequestQuit()
	SendToGnbUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nil, 2)
	if len(gnbue.ReadChan) != 1 {
		t.Errorf("a message was queued for a context that is quitting")
	}

	// a full queue must not block either
	for len(gnbue.ReadChan) < cap(gnbue.ReadChan) {
		gnbue.ReadChan <- &common.DefaultMessage{}
	}
	SendToGnbUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nil, 3)
}
