// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package context

import "testing"

func registerTargets(dao *GnbUeDao, ids ...int64) []*GnbCpUe {
	var targets []*GnbCpUe
	for _, id := range ids {
		c := NewGnbCpUe(id, nil, nil)
		c.HandoverNonce = dao.NextHandoverNonce()
		dao.AddGnbCpUe(id, c)
		dao.EnqueueHandoverTarget(c)
		targets = append(targets, c)
	}
	return targets
}

// A request that does not name its target is matched in order, and a target whose UE quit is
// skipped, since nothing will ask for it.
func TestDequeueHandoverTargetSkipsCancelledTargets(t *testing.T) {
	dao := NewGnbUeDao()
	targets := registerTargets(dao, 1, 2)
	dao.RemoveGnbCpUe(1)

	if got := dao.DequeueHandoverTarget(); got != targets[1] {
		t.Errorf("got %v, want the live target", got)
	}
	if got := dao.DequeueHandoverTarget(); got != nil {
		t.Errorf("got %v, want an empty queue", got)
	}
}

// A request that names its target gets exactly that one, whatever else is queued before it and
// whether or not the others are still alive.
func TestTakePendingHandoverTargetMatchesByIdentity(t *testing.T) {
	dao := NewGnbUeDao()
	targets := registerTargets(dao, 1, 2, 3)
	dao.RemoveGnbCpUe(1) // its UE quit; its request, if it ever comes, finds nobody

	if got := dao.TakePendingHandoverTarget(3, targets[2].HandoverNonce); got != targets[2] {
		t.Errorf("got %v, want target 3, which is not first in the queue", got)
	}
	if got := dao.TakePendingHandoverTarget(1, targets[0].HandoverNonce); got != nil {
		t.Errorf("got %v for a target that is gone, want nil and not another UE's target", got)
	}
	if got := dao.TakePendingHandoverTarget(2, targets[1].HandoverNonce); got != targets[1] {
		t.Errorf("got %v, want target 2, which must still be queued", got)
	}
	if got := dao.TakePendingHandoverTarget(2, targets[1].HandoverNonce); got != nil {
		t.Errorf("a target must be taken only once, got %v", got)
	}
}

func TestHandoverTargetTokenRoundTrip(t *testing.T) {
	for _, id := range []int64{0, 1, 4096, 1 << 40} {
		got, nonce, ok := ParseHandoverTargetToken(HandoverTargetToken(id, 77))
		if !ok || got != id || nonce != 77 {
			t.Errorf("id %d came back as %d, nonce %d, %v", id, got, nonce, ok)
		}
	}
	// the placeholder that is sent when the target is not known, and other implementations' content
	for _, container := range [][]byte{nil, {0x00}, []byte("some other container"), make([]byte, 20)} {
		if id, _, ok := ParseHandoverTargetToken(container); ok {
			t.Errorf("%v was taken for a token naming %d", container, id)
		}
	}
}

// RAN UE NGAP IDs are reused once a UE quits. A delayed request for the old registration must not
// take the target that was registered later with the same ID.
func TestTakePendingHandoverTargetIgnoresReusedId(t *testing.T) {
	dao := NewGnbUeDao()
	old := registerTargets(dao, 5)[0]
	oldNonce := old.HandoverNonce
	dao.RemoveGnbCpUe(5)
	replacement := registerTargets(dao, 5)[0] // same ID, another UE

	if got := dao.TakePendingHandoverTarget(5, oldNonce); got != nil {
		t.Errorf("the request of the old registration took %v", got)
	}
	if got := dao.TakePendingHandoverTarget(5, replacement.HandoverNonce); got != replacement {
		t.Errorf("got %v, want the replacement for its own request", got)
	}
}

// Targets whose request never comes must not pile up in the small queue until a registration is
// dropped.
func TestCancelledHandoverTargetsDoNotFillTheQueue(t *testing.T) {
	dao := NewGnbUeDao()
	live := registerTargets(dao, 1000)[0]
	for id := int64(1); id <= int64(3*cap(dao.pendingHandoverTargets)); id++ {
		registerTargets(dao, id)
		dao.RemoveGnbCpUe(id)
	}

	late := registerTargets(dao, 2000)[0]
	if got := dao.DequeueHandoverTarget(); got != live {
		t.Errorf("got %v, want the first live target, in order", got)
	}
	if got := dao.DequeueHandoverTarget(); got != late {
		t.Errorf("got %v, want the target registered after all the cancelled ones", got)
	}
}

// A user-plane context asks blocking sends to give way once, however often it is told to.
func TestGnbUpUeRequestQuitIsIdempotent(t *testing.T) {
	ue := NewGnbUpUe(1, 2, nil)
	select {
	case <-ue.QuitRequested():
		t.Fatal("quit requested before anyone asked")
	default:
	}
	ue.RequestQuit()
	ue.RequestQuit()
	select {
	case <-ue.QuitRequested():
	default:
		t.Error("quit was not requested")
	}
	(&GnbUpUe{}).RequestQuit() // a context that was not made by NewGnbUpUe must not panic
}
