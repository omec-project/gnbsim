// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/openapi/models"
	"github.com/sirupsen/logrus"
)

// profile names
const (
	REGISTER                string = "register"
	PDU_SESS_EST            string = "pdusessest"
	DEREGISTER              string = "deregister"
	AN_RELEASE              string = "anrelease"
	UE_TRIGG_SERVICE_REQ    string = "uetriggservicereq"
	NW_TRIGG_UE_DEREG       string = "nwtriggeruedereg"
	UE_REQ_PDU_SESS_RELEASE string = "uereqpdusessrelease"
	NW_REQ_PDU_SESS_RELEASE string = "nwreqpdusessrelease"
	CUSTOM_PROCEDURE        string = "custom"
)

const PER_USER_TIMEOUT uint32 = 100 // seconds
var SummaryChan = make(chan common.InterfaceMessage)

type ProcedureEventsDetails struct {
	Events map[common.EventType]common.EventType
}

var (
	ProceduresMap map[common.ProcedureType]*ProcedureEventsDetails
	ProfileMap    map[string]*Profile
)

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
	TrigEventsChan chan *common.ProfileMessage  // Receiving Events from the REST interface
	WriteSimChan   chan common.InterfaceMessage // Sending events to SIMUE -  start proc and proc parameters
	ReadChan       chan *common.ProfileMessage  // Sending events to profile

	Log *logrus.Entry

	CurrentItr       string // used only if UE is part of custom profile
	Repeat           int    // used only if UE is part of custom profile
	CurrentProcIndex int    // current procedure index. Used in custom profile
	Procedure        common.ProcedureType
}

type Profile struct {
	ProfileType    string         `yaml:"profileType" json:"profileType"`
	Name           string         `yaml:"profileName" json:"profileName"`
	GnbName        string         `yaml:"gnbName" json:"gnbName"`
	StartImsi      string         `yaml:"startImsi" json:"startImsi"`
	DefaultAs      string         `yaml:"defaultAs" json:"defaultAs"`
	Key            string         `yaml:"key" json:"key"`
	Opc            string         `yaml:"opc" json:"opc"`
	SeqNum         string         `yaml:"sequenceNumber" json:"sequenceNumber"`
	Dnn            string         `yaml:"dnn" json:"dnn"`
	StartIteration string         `yaml:"startiteration" json:"startiteration"`
	Plmn           *models.PlmnId `yaml:"plmnId" json:"plmnId"`
	SNssai         *models.Snssai `yaml:"sNssai" json:"sNssai"`
	Log            *logrus.Entry

	// Profile routine reads messages from other entities on this channel
	// Entities can be SimUe, Main routine.
	ReadChan chan *common.ProfileMessage

	Iterations  []*Iterations `yaml:"iterations"`
	PIterations map[string]*PIterations
	PSimUe      map[string]*ProfileUeContext
	Procedures  []common.ProcedureType

	Imsi           int    // StartImsi in int
	UeCount        int    `yaml:"ueCount" json:"ueCount"`
	DataPktCount   int    `yaml:"dataPktCount" json:"dataPktCount"`
	DataPktInt     int    `yaml:"dataPktInterval" json:"dataPktInterval"`
	PerUserTimeout uint32 `yaml:"perUserTimeout" json:"perUserTimeout"`
	Enable         bool   `yaml:"enable" json:"enable"`
	ExecInParallel bool   `yaml:"execInParallel" json:"execInParallel"`
	StepTrigger    bool   `yaml:"stepTrigger" json:"stepTrigger"`
	RetransMsg     bool   `yaml:"retransMsg" json:"retransMsg"`
}

func init() {
	ProceduresMap = make(map[common.ProcedureType]*ProcedureEventsDetails)
	ProfileMap = make(map[string]*Profile)
	initProcedureEventMap()
}

// predefined profiles
func initProcedureEventMap() {
	proc1 := ProcedureEventsDetails{}

	// common.REGISTRATION_PROCEDURE:
	proc1.Events = map[common.EventType]common.EventType{
		common.REG_REQUEST_EVENT:     common.AUTH_REQUEST_EVENT,
		common.AUTH_REQUEST_EVENT:    common.AUTH_RESPONSE_EVENT,
		common.SEC_MOD_COMMAND_EVENT: common.SEC_MOD_COMPLETE_EVENT,
		common.REG_ACCEPT_EVENT:      common.REG_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:    common.QUIT_EVENT,
	}
	ProceduresMap[common.REGISTRATION_PROCEDURE] = &proc1

	// common.PDU_SESSION_ESTABLISHMENT_PROCEDURE:
	proc2 := ProcedureEventsDetails{}
	proc2.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_EST_REQUEST_EVENT: common.PDU_SESS_EST_ACCEPT_EVENT,
		common.PDU_SESS_EST_ACCEPT_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	ProceduresMap[common.PDU_SESSION_ESTABLISHMENT_PROCEDURE] = &proc2

	// common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
	proc3 := ProcedureEventsDetails{}
	proc3.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_REL_REQUEST_EVENT: common.PDU_SESS_REL_COMMAND_EVENT,
		common.PDU_SESS_REL_COMMAND_EVENT: common.PDU_SESS_REL_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	ProceduresMap[common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE] = &proc3

	// common.UE_INITIATED_DEREGISTRATION_PROCEDURE:
	proc4 := ProcedureEventsDetails{}
	proc4.Events = map[common.EventType]common.EventType{
		common.DEREG_REQUEST_UE_ORIG_EVENT: common.DEREG_ACCEPT_UE_ORIG_EVENT,
		common.PROFILE_PASS_EVENT:          common.QUIT_EVENT,
	}
	ProceduresMap[common.UE_INITIATED_DEREGISTRATION_PROCEDURE] = &proc4

	// common.AN_RELEASE_PROCEDURE:
	proc5 := ProcedureEventsDetails{}
	proc5.Events = map[common.EventType]common.EventType{
		common.TRIGGER_AN_RELEASE_EVENT: common.CONNECTION_RELEASE_REQUEST_EVENT,
		common.PROFILE_PASS_EVENT:       common.QUIT_EVENT,
	}
	ProceduresMap[common.AN_RELEASE_PROCEDURE] = &proc5

	// common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE:
	proc6 := ProcedureEventsDetails{}
	proc6.Events = map[common.EventType]common.EventType{
		common.SERVICE_REQUEST_EVENT: common.SERVICE_ACCEPT_EVENT,
		common.PROFILE_PASS_EVENT:    common.QUIT_EVENT,
	}
	ProceduresMap[common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE] = &proc6

	// common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE:
	proc7 := ProcedureEventsDetails{}
	proc7.Events = map[common.EventType]common.EventType{
		common.DEREG_REQUEST_UE_TERM_EVENT: common.DEREG_ACCEPT_UE_TERM_EVENT,
		common.PROFILE_PASS_EVENT:          common.QUIT_EVENT,
	}
	ProceduresMap[common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE] = &proc7

	// common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE:
	proc8 := ProcedureEventsDetails{}
	proc8.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_REL_COMMAND_EVENT: common.PDU_SESS_REL_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	ProceduresMap[common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE] = &proc8

	// common.USER_DATA_PKT_GENERATION_PROCEDURE:
	proc9 := ProcedureEventsDetails{}
	proc9.Events = map[common.EventType]common.EventType{
		common.PROFILE_PASS_EVENT: common.QUIT_EVENT,
	}
	ProceduresMap[common.USER_DATA_PKT_GENERATION_PROCEDURE] = &proc9
}

func (profile *Profile) Init() error {
	profile.ReadChan = make(chan *common.ProfileMessage)
	profile.PSimUe = make(map[string]*ProfileUeContext)
	profile.Log = logger.ProfileLog.WithField(logger.FieldProfile, profile.Name)
	if profile.DataPktCount == 0 {
		profile.DataPktCount = 5 // default
	}
	if profile.DefaultAs == "" {
		profile.DefaultAs = "192.168.250.1" // default destination for AIAB
	}

	if profile.PerUserTimeout == 0 {
		profile.PerUserTimeout = PER_USER_TIMEOUT
	}

	err := initProcedureList(profile)
	if err != nil {
		return err
	}

	ProfileMap[profile.Name] = profile
	profile.Log.Traceln("profile initialized ", profile.Name, ", Enable ", profile.Enable)
	return nil
}

func initProcedureList(profile *Profile) error {
	switch profile.ProfileType {
	case REGISTER:
		profile.Procedures = []common.ProcedureType{common.REGISTRATION_PROCEDURE}
	case PDU_SESS_EST:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
		}
	case DEREGISTER:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
		}
	case AN_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.AN_RELEASE_PROCEDURE,
		}
	case UE_TRIGG_SERVICE_REQ:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.AN_RELEASE_PROCEDURE,
			common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
		}
	case NW_TRIGG_UE_DEREG:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.NW_TRIGGERED_UE_DEREGISTRATION_PROCEDURE,
		}
	case UE_REQ_PDU_SESS_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.UE_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE,
		}
	case NW_REQ_PDU_SESS_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.NW_REQUESTED_PDU_SESSION_RELEASE_PROCEDURE,
		}

	case CUSTOM_PROCEDURE:
		// Custom Profiles do not have prefdefined procedure list
		return nil

	default:
		return fmt.Errorf("profile type not supported: %v", profile.ProfileType)
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
		itp := p.PIterations[pCtx.CurrentItr]
		pCtx.Log.Infoln("Current Iteration map - ", itp)
		if itp.WaitMap[pCtx.CurrentProcIndex] != 0 {
			time.Sleep(time.Millisecond * time.Duration(itp.WaitMap[pCtx.CurrentProcIndex]))
		}
		nextProcIndex := pCtx.CurrentProcIndex + 1
		var found bool
		nextProcedure, found = itp.ProcMap[nextProcIndex]
		if found {
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
			nextProcedure = itp.ProcMap[1]
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
			nextProcedure = nItr.ProcMap[1]
			pCtx.Procedure = nextProcedure
			return nextProcedure
		}
	}
	pCtx.Log.Infoln("Nothing more to execute for UE")
	return nextProcedure
}
