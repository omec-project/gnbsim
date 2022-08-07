// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"

	"github.com/omec-project/openapi/models"
	"github.com/sirupsen/logrus"
)

const PER_USER_TIMEOUT uint32 = 100 //seconds
var SummaryChan = make(chan common.InterfaceMessage)

type Profile struct {
	ProfileType    string         `yaml:"profileType" json:"profileType"`
	Name           string         `yaml:"profileName" json:"profileName"`
	Enable         bool           `yaml:"enable" json:"enable"`
	GnbName        string         `yaml:"gnbName" json:"gnbName"`
	StartImsi      string         `yaml:"startImsi" json:"startImsi"`
	UeCount        int            `yaml:"ueCount" json:"ueCount"`
	Plmn           *models.PlmnId `yaml:"plmnId" json:"plmnId"`
	DataPktCount   int            `yaml:"dataPktCount" json:"dataPktCount"`
	PerUserTimeout uint32         `yaml:"perUserTimeout" json:"perUserTimeout"`
	DefaultAs      string         `yaml:"defaultAs" json:"defaultAs"`
	Key            string         `yaml:"key" json:"key"`
	Opc            string         `yaml:"opc" json:"opc"`
	SeqNum         string         `yaml:"sequenceNumber" json:"sequenceNumber"`
	Dnn            string         `yaml:"dnn" json:"dnn"`
	SNssai         *models.Snssai `yaml:"sNssai" json:"sNssai"`

	Events     map[common.EventType]common.EventType
	Procedures []common.ProcedureType

	// Profile routine reads messages from other entities on this channel
	// Entities can be SimUe, Main routine.
	ReadChan chan *common.ProfileMessage

	/* logger */
	Log *logrus.Entry
}

func (profile *Profile) Init() {
	profile.ReadChan = make(chan *common.ProfileMessage)
	profile.Log = logger.ProfileLog.WithField(logger.FieldProfile, profile.Name)

	profile.Log.Traceln("profile initialized ", profile.Name, ", Enable ", profile.Enable)
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
