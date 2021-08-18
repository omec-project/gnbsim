// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	intfc "gnbsim/interfacecommon"
)

type GnbUe struct {
	// Should IMSI be stored in GnbUe
	Supi        string
	GnbUeNgapId int64
	AmfUeNgapId int64
	Amf         *GnbAmf
	Gnb         *GNodeB
	// TODO MME details

	// GnbUe writes messages to UE on this channel
	WriteUeChan chan<- *intfc.UuMessage

	// GnbUe reads messages from all other workers and UE on this channel
	ReadChan chan intfc.InterfaceMessage
}

func NewGnbUe(ngapId int64, gnb *GNodeB, amf *GnbAmf) *GnbUe {
	gnbue := GnbUe{}
	gnbue.GnbUeNgapId = ngapId
	gnbue.Amf = amf
	gnbue.Gnb = gnb
	gnbue.ReadChan = make(chan intfc.InterfaceMessage)
	return &gnbue
}
