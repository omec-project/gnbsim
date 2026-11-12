// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"github.com/omec-project/gnbsim/common"
	simueCtx "github.com/omec-project/gnbsim/simue/context"
)

func SendToSimUe(simUe *simueCtx.SimUe, event common.EventType, proc common.ProcedureType) {
	msg := &common.ProfileMessage{}
	msg.Event = event
	msg.Proc = proc
	// a SimUe that is quitting reads no more, and nobody is waiting for this any longer
	select {
	case simUe.ReadChan <- msg:
	case <-simUe.QuitRequested():
	}
}
