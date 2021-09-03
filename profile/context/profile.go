// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/logger"

	"github.com/sirupsen/logrus"
)

type Profile struct {
	Name       string
	Events     map[common.EventType]common.EventType
	Procedures []common.ProcedureType
	GnbName    string
	StartImsi  string
	UeCount    uint32

	// Profile routine reads messages from other entities on this channel
	// Entities can be SimUe, Main routine.
	ReadChan chan *common.ProfileMessage

	/* logger */
	Log *logrus.Entry
}

func NewProfile(name string) *Profile {
	profile := Profile{}
	profile.Name = name
	profile.GnbName = "gnodeb1"
	profile.StartImsi = "imsi-2089300007487"
	profile.UeCount = 1
	profile.ReadChan = make(chan *common.ProfileMessage)
	profile.Log = logger.ProfileLog.WithField(logger.FieldProfile, name)

	profile.Log.Traceln("Created new context")

	return &profile
}

func (p *Profile) GetNextEvent(currentEvent common.EventType) (common.EventType, error) {
	var err error
	nextEvent, ok := p.Events[currentEvent]
	if !ok {
		err = fmt.Errorf("event %v not configured in event map", currentEvent)
	}
	return nextEvent, err
}

func (p *Profile) CheckCurrentEvent(triggerEvent, recvEvent common.EventType) (err error) {
	expected, ok := p.Events[triggerEvent]
	if !ok {
		err = fmt.Errorf("triggering event %v not configured in event map",
			triggerEvent)
	} else if recvEvent != expected {
		err = fmt.Errorf("triggering event:%v, expected event:%v, received event:%v",
			triggerEvent, expected, recvEvent)
	}
	return err
}

func (p *Profile) GetFirstProcedure() common.ProcedureType {
	if len(p.Procedures) == 0 {
		p.Log.Fatalln("Procedure List Empty")
	}
	return p.Procedures[0]
}

func (p *Profile) GetNextProcedure(currentProcedure common.ProcedureType) common.ProcedureType {
	length := len(p.Procedures)
	var nextProcedure common.ProcedureType

	for i, procedure := range p.Procedures {
		if currentProcedure == procedure {
			// Checking if i is not the last index
			if length > (i + 1) {
				nextProcedure = p.Procedures[i+1]
				break
			}

			p.Log.Infoln("No more procedures left")
		}
	}

	return nextProcedure
}
