// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"log"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"

	"github.com/omec-project/openapi/models"
	"github.com/sirupsen/logrus"
)

const PER_USER_TIMEOUT uint32 = 100 //seconds
var SummaryChan = make(chan common.InterfaceMessage)

type ProcedureEventsDetails struct {
	Events map[common.EventType]common.EventType
}

var ProceduresMap map[common.ProcedureType]*ProcedureEventsDetails
var ProfileMap map[string]*Profile

type PIterations struct {
	Name    string
	ProcMap map[int]common.ProcedureType
	WaitMap map[int]int
	NextItr string
	Repeat  int
}

type Iterations struct {
	Name    string `yaml:"name"`
	First   string `yaml:"1"`
	Second  string `yaml:"2"`
	Third   string `yaml:"3"`
	Fourth  string `yaml:"4"`
	Fifth   string `yaml:"5"`
	Sixth   string `yaml:"6"`
	Seventh string `yaml:"7"`
	Next    string `yaml:"next"`
	Repeat  int    `yaml:"repeat"`
}

type ProfileUeContext struct {
	TrigEventsChan   chan *common.ProfileMessage  // Receiving Events from the REST interface
	WriteSimChan     chan common.InterfaceMessage // Sending events to SIMUE -  start proc and proc parameters
	ReadChan         chan *common.ProfileMessage  // simUe to profile ?
	Repeat           int                          // used only if UE is part of custom profile
	CurrentItr       string                       // used only if UE is part of custom profile
	CurrentProcIndex int                          // current procedure index. Used in custom profile
	Procedure        common.ProcedureType

	/* logger */
	Log *logrus.Entry
}

type Profile struct {
	ProfileType    string         `yaml:"profileType" json:"profileType"`
	Name           string         `yaml:"profileName" json:"profileName"`
	Enable         bool           `yaml:"enable" json:"enable"`
	GnbName        string         `yaml:"gnbName" json:"gnbName"`
	StartImsi      string         `yaml:"startImsi" json:"startImsi"`
	Imsi           int            // StartImsi in int
	UeCount        int            `yaml:"ueCount" json:"ueCount"`
	Plmn           *models.PlmnId `yaml:"plmnId" json:"plmnId"`
	DataPktCount   int            `yaml:"dataPktCount" json:"dataPktCount"`
	DataPktInt     int            `yaml:"dataPktInterval" json:"dataPktInterval"`
	PerUserTimeout uint32         `yaml:"perUserTimeout" json:"perUserTimeout"`
	DefaultAs      string         `yaml:"defaultAs" json:"defaultAs"`
	Key            string         `yaml:"key" json:"key"`
	Opc            string         `yaml:"opc" json:"opc"`
	SeqNum         string         `yaml:"sequenceNumber" json:"sequenceNumber"`
	Dnn            string         `yaml:"dnn" json:"dnn"`
	SNssai         *models.Snssai `yaml:"sNssai" json:"sNssai"`
	ExecInParallel bool           `yaml:"execInParallel" json:"execInParallel"`
	StepTrigger    bool           `yaml:"stepTrigger" json:"stepTrigger"`
	StartIteration string         `yaml:"startiteration" json:"startiteration"`
	Iterations     []*Iterations  `yaml:"iterations"`

	PIterations map[string]*PIterations
	Procedures  []common.ProcedureType

	// Profile routine reads messages from other entities on this channel
	// Entities can be SimUe, Main routine.
	ReadChan chan *common.ProfileMessage

	PSimUe map[string]*ProfileUeContext

	/* logger */
	Log *logrus.Entry
}

func init() {
	ProceduresMap = make(map[common.ProcedureType]*ProcedureEventsDetails)
	ProfileMap = make(map[string]*Profile)
}

func (profile *Profile) Init() {
	profile.ReadChan = make(chan *common.ProfileMessage)
	profile.PSimUe = make(map[string]*ProfileUeContext)
	profile.Log = logger.ProfileLog.WithField(logger.FieldProfile, profile.Name)
	if profile.DataPktCount == 0 {
		profile.DataPktCount = 5 // default
	}
	if profile.DefaultAs == "" {
		profile.DefaultAs = "192.168.250.1" // default destination for AIAB
	}
	ProfileMap[profile.Name] = profile
	profile.Log.Traceln("profile initialized ", profile.Name, ", Enable ", profile.Enable)
}

// enable step trigger only if execParallel is enabled in profile
func SendStepEventProfile(name string) error {
	profile, found := ProfileMap[name]
	if found == false {
		err := fmt.Errorf("unknown profile:%s", profile)
		log.Println(err)
		return err
	}
	if profile.ExecInParallel == false {
		err := fmt.Errorf("ExecInParallel should be true if step profile needs to be executed")
		log.Println(err)
		return err
	}
	msg := &common.ProfileMessage{}
	// msg.Supi =
	// msg.ProcedureType =
	msg.Event = common.PROFILE_STEP_EVENT
	for _, ctx := range profile.PSimUe {
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - start")
		ctx.TrigEventsChan <- msg
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - end")
	}
	return nil
}

func SendAddNewCallsEventProfile(name string, number int32) error {
	profile, found := ProfileMap[name]
	if found == false {
		err := fmt.Errorf("unknown profile:%s", profile)
		return err
	}
	msg := &common.ProfileMessage{}
	msg.Event = common.PROFILE_ADDCALLS_EVENT
	var i int32
	for i = 0; i < number; i++ {
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - start")
		profile.ReadChan <- msg
		profile.Log.Traceln("profile ", profile, ", writing on trig channel - end")
	}
	return nil
}

func (p *Profile) GetNextEvent(Procedure common.ProcedureType, currentEvent common.EventType) (common.EventType, error) {
	var err error
	proc := ProceduresMap[Procedure]

	nextEvent, ok := proc.Events[currentEvent]
	if !ok {
		err = fmt.Errorf("event %v not configured in event map", currentEvent)
	}
	return nextEvent, err
}

func (p *Profile) CheckCurrentEvent(Procedure common.ProcedureType, triggerEvent, recvEvent common.EventType) (err error) {
	proc := ProceduresMap[Procedure]
	expected, ok := proc.Events[triggerEvent]
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

/*
func ChangeProcedure(ue *simuectx.SimUe) {
	//no procedure to be executed. Send PROFILE_PASS_EVENT
	SendToProfile(ue, common.PROFILE_PASS_EVENT, nil)
	evt, err := ue.ProfileCtx.GetNextEvent(ue.Procedure, common.PROFILE_PASS_EVENT)
	// This is suppose to be last event..why do we care to get return error ?
	if err != nil {
		ue.Log.Errorln("GetNextEvent failed:", err)
		return
	}
	if evt == common.QUIT_EVENT {
		msg := &common.DefaultMessage{}
		msg.Event = common.QUIT_EVENT
		ue.ReadChan <- msg
	}
	return
}*/

func (p *Profile) GetNextProcedure(pCtx *ProfileUeContext, currentProcedure common.ProcedureType) common.ProcedureType {
	length := len(p.Procedures)
	var nextProcedure common.ProcedureType

	// check if predefined profiles.
	if len(p.Iterations) == 0 {
		if currentProcedure == 0 {
			proc := p.GetFirstProcedure()
			return proc
		}
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

	// check if custom Profile
	if len(p.Iterations) > 0 {
		pCtx.Log.Infoln("Current UE iteration ", pCtx.CurrentItr)
		pCtx.Log.Infoln("Current UE procedure index  ", pCtx.CurrentProcIndex)
		itp, found := p.PIterations[pCtx.CurrentItr]
		pCtx.Log.Infoln("Current Iteration map - ", itp)
		if itp.WaitMap[pCtx.CurrentProcIndex] != 0 {
			time.Sleep(time.Second * time.Duration(itp.WaitMap[pCtx.CurrentProcIndex]))
		}
		nextProcIndex := pCtx.CurrentProcIndex + 1
		nextProcedure, found := itp.ProcMap[nextProcIndex]
		if found == true {
			pCtx.Log.Infof("Next Procedure Index %v and next Procedure = %v ", nextProcIndex, nextProcedure)
			pCtx.Procedure = nextProcedure
			pCtx.CurrentProcIndex = nextProcIndex
			pCtx.Log.Infoln("Updated procedure to", nextProcedure)
			return nextProcedure
		}
		if pCtx.Repeat > 0 {
			pCtx.Repeat = pCtx.Repeat - 1
			pCtx.Log.Infoln("Repeat current iteration : ", itp.Name, ", Repeat Count ", pCtx.Repeat)
			pCtx.CurrentProcIndex = 1
			nextProcedure := itp.ProcMap[1]
			pCtx.Procedure = nextProcedure
			return nextProcedure
		}
		pCtx.Log.Infoln("Iteration Complete ", pCtx.CurrentItr)
		nextItr := itp.NextItr
		if nextItr != "quit" {
			nItr := p.PIterations[nextItr]
			pCtx.Log.Infoln("Going to next iteration ", nItr.Name)
			pCtx.CurrentItr = nItr.Name
			pCtx.CurrentProcIndex = 1
			pCtx.Repeat = nItr.Repeat
			nextProcedure := nItr.ProcMap[1]
			pCtx.Procedure = nextProcedure
			return nextProcedure
		}
	}
	pCtx.Log.Infoln("Nothing more to execute for UE")
	return nextProcedure
}
