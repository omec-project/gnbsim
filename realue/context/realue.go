// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	intfc "gnbsim/interfacecommon"

	"github.com/free5gc/nas/security"
	"github.com/free5gc/openapi/models"
)

// RealUe represents a Real UE
type RealUe struct {
	Supi               string
	Imsi               string
	Guti               string
	ULCount            security.Count
	DLCount            security.Count
	CipheringAlg       uint8
	IntegrityAlg       uint8
	KnasEnc            [16]uint8
	KnasInt            [16]uint8
	Kamf               []uint8
	AuthenticationSubs models.AuthenticationSubscription

	//RealUe writes messages to SimUE on this channel
	WriteSimUeChan chan *intfc.UuMessage

	//RealUe reads messages from SimUE on this channel
	ReadChan chan *intfc.UuMessage
}

func NewRealUeContext(supi string, cipheringAlg, integrityAlg uint8,
	simuechan chan *intfc.UuMessage) *RealUe {

	ue := RealUe{}
	ue.Supi = supi
	ue.CipheringAlg = cipheringAlg
	ue.IntegrityAlg = integrityAlg
	ue.WriteSimUeChan = simuechan
	ue.ReadChan = make(chan *intfc.UuMessage)
	return &ue
}
