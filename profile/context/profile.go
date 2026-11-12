// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/openapi/v2/models"
	"go.uber.org/zap"
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
	N2_HANDOVER             string = "n2handover"
)

const (
	quit = "quit"
)

const PER_USER_TIMEOUT uint32 = 100 // seconds
var SummaryChan = make(chan common.InterfaceMessage)

type ProcedureEventsDetails struct {
	Events map[common.EventType]common.EventType
}

var (
	ProceduresMap map[common.ProcedureType]*ProcedureEventsDetails
	ProfileMap    map[string]*Profile
	// ProfileMapMu guards ProfileMap, which is written by profile initialisation and
	// read by the REST handlers on different goroutines.
	ProfileMapMu sync.RWMutex
)

// Publish makes the profile reachable by name (REST step and add-calls requests). It must be
// called once the profile is fully initialised, i.e. after its UE contexts have been created.
func (p *Profile) Publish() {
	ProfileMapMu.Lock()
	ProfileMap[p.Name] = p
	ProfileMapMu.Unlock()
}

// GetProfile returns the registered profile with the given name.
func GetProfile(name string) (*Profile, bool) {
	ProfileMapMu.RLock()
	defer ProfileMapMu.RUnlock()
	p, ok := ProfileMap[name]
	return p, ok
}

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
	// Quit asks the SimUe to tear down. It does not depend on the capacity of WriteSimChan, and
	// calling it more than once is harmless.
	Quit func()
	// Terminated is closed once the SimUe has finished tearing down and its SUPI is free.
	Terminated <-chan struct{}
	ReadChan   chan *common.ProfileMessage // Sending events to profile

	Log *zap.SugaredLogger

	CurrentItr       string // used only if UE is part of custom profile
	Repeat           int    // used only if UE is part of custom profile
	CurrentProcIndex int    // current procedure index. Used in custom profile
	Procedure        common.ProcedureType
}

type Profile struct {
	Plmn                         *models.PlmnId `yaml:"plmnId" json:"plmnId"`
	PSimUe                       map[string]*ProfileUeContext
	PIterations                  map[string]*PIterations
	ReadChan                     chan *common.ProfileMessage
	Log                          *zap.SugaredLogger
	SNssai                       *models.Snssai `yaml:"sNssai" json:"sNssai"`
	SeqNum                       string         `yaml:"sequenceNumber" json:"sequenceNumber"`
	Name                         string         `yaml:"profileName" json:"profileName"`
	ProfileType                  string         `yaml:"profileType" json:"profileType"`
	Dnn                          string         `yaml:"dnn" json:"dnn"`
	StartIteration               string         `yaml:"startiteration" json:"startiteration"`
	Key                          string         `yaml:"key" json:"key"`
	DefaultAs                    string         `yaml:"defaultAs" json:"defaultAs"`
	StartImsi                    string         `yaml:"startImsi" json:"startImsi"`
	TargetGnbName                string         `yaml:"targetGnbName" json:"targetGnbName"`
	Opc                          string         `yaml:"opc" json:"opc"`
	GnbName                      string         `yaml:"gnbName" json:"gnbName"`
	Procedures                   []common.ProcedureType
	Iterations                   []*Iterations `yaml:"iterations"`
	Imsi                         int
	UeCount                      int          `yaml:"ueCount" json:"ueCount"`
	DataPktCount                 int          `yaml:"dataPktCount" json:"dataPktCount"`
	DataPktInt                   int          `yaml:"dataPktInterval" json:"dataPktInterval"`
	PSimUeMu                     sync.RWMutex `yaml:"-" json:"-"`
	PerUserTimeout               uint32       `yaml:"perUserTimeout" json:"perUserTimeout"`
	Enable                       bool         `yaml:"enable" json:"enable"`
	ExecInParallel               bool         `yaml:"execInParallel" json:"execInParallel"`
	StepTrigger                  bool         `yaml:"stepTrigger" json:"stepTrigger"`
	RetransMsg                   bool         `yaml:"retransMsg" json:"retransMsg"`
	WithholdModificationComplete bool         `yaml:"withholdModificationComplete" json:"withholdModificationComplete"`
	ModificationRequestType      uint8        `yaml:"modificationRequestType" json:"modificationRequestType"`
	OmitModificationRequestType  bool         `yaml:"omitModificationRequestType" json:"omitModificationRequestType"`
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

	// common.NW_PDU_SESSION_MODIFICATION_PROCEDURE:
	// The network sends the command unprompted, so the procedure begins on receipt rather than on
	// anything the UE does. Withholding the complete is a profile option rather than a second map:
	// the network's retransmission behaviour is the thing under test there, not the UE's.
	procNwPduSessMod := ProcedureEventsDetails{}
	procNwPduSessMod.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_MOD_COMMAND_EVENT: common.PDU_SESS_MOD_COMPLETE_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	ProceduresMap[common.NW_PDU_SESSION_MODIFICATION_PROCEDURE] = &procNwPduSessMod

	// common.UE_REQUESTED_PDU_SESSION_MODIFICATION_PROCEDURE:
	// The core refuses every UE-requested modification, so the reject is the expected outcome and
	// the procedure ends there. There is no accept path to map, because there is none to reach.
	procUePduSessMod := ProcedureEventsDetails{}
	procUePduSessMod.Events = map[common.EventType]common.EventType{
		common.PDU_SESS_MOD_REQUEST_EVENT: common.PDU_SESS_MOD_REJECT_EVENT,
		common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
	}
	ProceduresMap[common.UE_REQUESTED_PDU_SESSION_MODIFICATION_PROCEDURE] = &procUePduSessMod

	// common.USER_DATA_PKT_GENERATION_PROCEDURE:
	proc9 := ProcedureEventsDetails{}
	proc9.Events = map[common.EventType]common.EventType{
		common.PROFILE_PASS_EVENT: common.QUIT_EVENT,
	}
	ProceduresMap[common.USER_DATA_PKT_GENERATION_PROCEDURE] = &proc9

	proc10 := ProcedureEventsDetails{}
	proc10.Events = map[common.EventType]common.EventType{
		common.TRIGGER_HO_EVENT:   common.HO_COMMAND_EVENT,
		common.PROFILE_PASS_EVENT: common.QUIT_EVENT,
	}
	ProceduresMap[common.N2_HANDOVER_PROCEDURE] = &proc10
}

// BuildIterations converts the iteration definitions of a custom profile into PIterations and
// checks that every reference (start iteration, next iteration) resolves. It is a no-op for
// profiles without iterations. It runs for profiles from the configuration file and from the
// REST API alike, since both are executed through GetNextProcedure.
func (p *Profile) BuildIterations() error {
	if len(p.Iterations) == 0 {
		return nil
	}
	p.PIterations = make(map[string]*PIterations)
	for _, it := range p.Iterations {
		if it == nil {
			return fmt.Errorf("profile %q: empty iteration definition", p.Name)
		}
		if len(it.Next) == 0 {
			it.Next = quit // default value
		}
		pIter := &PIterations{Name: it.Name, NextItr: it.Next, Repeat: it.Repeat}
		pIter.ProcMap = make(map[int]common.ProcedureType)
		pIter.WaitMap = make(map[int]int)
		pIter.WaitMap[0] = 0
		steps := []string{it.First, it.Second, it.Third, it.Fourth, it.Fifth, it.Sixth, it.Seventh}
		for i, step := range steps {
			if len(step) == 0 {
				continue
			}
			proc, wait, err := parseIterationStep(step)
			if err != nil {
				return fmt.Errorf("iteration %q step %d: %v", it.Name, i+1, err)
			}
			pIter.ProcMap[i+1] = proc
			pIter.WaitMap[i+1] = wait
		}
		p.PIterations[it.Name] = pIter
	}
	// every reference must resolve, otherwise the profile dereferences a nil iteration at run time
	if _, ok := p.PIterations[p.StartIteration]; !ok {
		return fmt.Errorf("custom profile %q: startiteration %q is not a defined iteration", p.Name, p.StartIteration)
	}
	for _, pit := range p.PIterations {
		if pit.NextItr == quit {
			continue
		}
		if _, ok := p.PIterations[pit.NextItr]; !ok {
			return fmt.Errorf("custom profile %q: iteration %q has unknown next iteration %q", p.Name, pit.Name, pit.NextItr)
		}
	}
	return nil
}

// parseIterationStep parses a custom profile step of the form "<PROCEDURE-NAME> <wait-ms>".
func parseIterationStep(step string) (common.ProcedureType, int, error) {
	fields := strings.Fields(step)
	if len(fields) != 2 {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("expected \"<procedure> <wait>\", got %q", step)
	}
	proc := common.GetProcId(fields[0])
	if proc == common.UNKNOWN_PROCEDURE {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("unknown procedure %q", fields[0])
	}
	wait, err := strconv.Atoi(fields[1])
	if err != nil {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("value is not converted to integer: %v", err)
	}
	if wait < 0 {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("wait must not be negative, got %d", wait)
	}
	return proc, wait, nil
}

// maxImsiDigits is the longest IMSI there is: "The number of digits in IMSI shall not exceed 15"
// (TS 23.003 clause 2.3).
const maxImsiDigits = 15

// validateStartImsi checks that the IMSIs of the profile, from startImsi on, are IMSIs: decimal
// digits only (TS 23.003 clause 2.3), at most 15 of them even for the last UE. An empty startImsi
// is left to InitProfile, which reports it.
func validateStartImsi(profile *Profile) error {
	if profile.StartImsi == "" {
		return nil
	}
	if len(profile.StartImsi) > maxImsiDigits {
		return fmt.Errorf("startImsi has %d digits, an IMSI has at most %d", len(profile.StartImsi), maxImsiDigits)
	}
	for _, r := range profile.StartImsi {
		if r < '0' || r > '9' {
			return fmt.Errorf("startImsi %q must consist of decimal digits only", profile.StartImsi)
		}
	}
	start, err := strconv.Atoi(profile.StartImsi) // at most 15 digits, so it fits
	if err != nil {
		return fmt.Errorf("startImsi %q: %v", profile.StartImsi, err)
	}
	if last := start + max(profile.UeCount, 1) - 1; len(strconv.Itoa(last)) > maxImsiDigits {
		return fmt.Errorf("startImsi %q and ueCount %d run past %d digits", profile.StartImsi, profile.UeCount, maxImsiDigits)
	}
	return nil
}

// SetUe registers the context of a UE.
func (p *Profile) SetUe(imsi string, ctx *ProfileUeContext) {
	p.PSimUeMu.Lock()
	p.PSimUe[imsi] = ctx
	p.PSimUeMu.Unlock()
}

// DeleteUe forgets the context of a UE.
func (p *Profile) DeleteUe(imsi string) {
	p.PSimUeMu.Lock()
	delete(p.PSimUe, imsi)
	p.PSimUeMu.Unlock()
}

// GetUe returns the context of a UE, or nil if there is none.
func (p *Profile) GetUe(imsi string) *ProfileUeContext {
	p.PSimUeMu.RLock()
	defer p.PSimUeMu.RUnlock()
	return p.PSimUe[imsi]
}

// Ues returns a snapshot of the UE contexts, safe to iterate while UEs are being added.
func (p *Profile) Ues() []*ProfileUeContext {
	p.PSimUeMu.RLock()
	defer p.PSimUeMu.RUnlock()
	ues := make([]*ProfileUeContext, 0, len(p.PSimUe))
	for _, ctx := range p.PSimUe {
		ues = append(ues, ctx)
	}
	return ues
}

// Init prepares the profile for execution. It does not publish it; see Publish.
func (profile *Profile) Init() error {
	profile.ReadChan = make(chan *common.ProfileMessage)
	profile.PSimUe = make(map[string]*ProfileUeContext)
	profile.Log = logger.ProfileLog.With(logger.FieldProfile, profile.Name)
	if profile.DataPktCount == 0 {
		profile.DataPktCount = 5 // default
	}
	if profile.DefaultAs == "" {
		profile.DefaultAs = "192.168.250.1" // default destination for AIAB
	}
	// the ICMP echo request that is sent after PDU session establishment is IPv4 only, and an
	// address that does not parse makes the packet builder panic
	if net.ParseIP(profile.DefaultAs).To4() == nil {
		return fmt.Errorf("defaultAs must be an IPv4 address, got %q", profile.DefaultAs)
	}

	if profile.PerUserTimeout == 0 {
		profile.PerUserTimeout = PER_USER_TIMEOUT
	}

	if profile.UeCount < 0 {
		return fmt.Errorf("ueCount must not be negative, got %d", profile.UeCount)
	}
	// a negative interval fires every timer at once, which turns the packet generator into a flood
	if profile.DataPktCount < 0 || profile.DataPktInt < 0 {
		return fmt.Errorf("dataPktCount and dataPktInterval must not be negative, got %d and %d",
			profile.DataPktCount, profile.DataPktInt)
	}
	if err := validateStartImsi(profile); err != nil {
		return err
	}

	err := initProcedureList(profile)
	if err != nil {
		return err
	}

	if err = profile.BuildIterations(); err != nil {
		return err
	}

	// Validate Dnn and SNssai for profiles with PDU session establishment
	if requiresPduSession(profile) {
		if profile.Dnn == "" {
			return fmt.Errorf("dnn is required for profile type '%s'", profile.ProfileType)
		}
		if profile.SNssai == nil {
			return fmt.Errorf("sNssai is required for profile type '%s'", profile.ProfileType)
		}
		if profile.SNssai.Sst == 0 {
			return fmt.Errorf("sNssai.sst is required for profile type '%s'", profile.ProfileType)
		}
	}

	profile.Log.Debugln("profile initialized", profile.Name, ", Enable", profile.Enable)
	return nil
}

// requiresPduSession checks if a profile requires PDU session establishment
// For predefined profiles, it checks profile.Procedures
// For custom profiles, it scans through profile.PIterations to check if any iteration contains PDU_SESSION_ESTABLISHMENT_PROCEDURE
func requiresPduSession(profile *Profile) bool {
	// Check predefined profiles
	if slices.Contains(profile.Procedures, common.PDU_SESSION_ESTABLISHMENT_PROCEDURE) {
		return true
	}

	// Check custom profiles by scanning through iterations
	for _, iteration := range profile.PIterations {
		for _, procedure := range iteration.ProcMap {
			if procedure == common.PDU_SESSION_ESTABLISHMENT_PROCEDURE {
				return true
			}
		}
	}

	return false
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

	case N2_HANDOVER:
		if profile.TargetGnbName == "" {
			return fmt.Errorf("targetGnbName is required for profile type '%s'", profile.ProfileType)
		}
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE, // verify UP on source gNB
			common.N2_HANDOVER_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE, // verify UP on target gNB
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

// GetFirstProcedure returns the first procedure of a predefined profile, or
// UNKNOWN_PROCEDURE if the list is empty, which callers treat as "nothing to execute".
func (p *Profile) GetFirstProcedure() common.ProcedureType {
	if len(p.Procedures) == 0 {
		p.Log.Errorln("Procedure List Empty")
		return common.UNKNOWN_PROCEDURE
	}
	return p.Procedures[0]
}

func (p *Profile) GetNextProcedure(pCtx *ProfileUeContext, currentProcedure common.ProcedureType) (common.ProcedureType, error) {
	length := len(p.Procedures)
	var nextProcedure common.ProcedureType

	// check if predefined profiles.
	if len(p.Iterations) == 0 {
		if currentProcedure == 0 {
			pCtx.CurrentProcIndex = 0
			return p.GetFirstProcedure(), nil
		}
		// Use CurrentProcIndex to find the exact position we are at so that
		// duplicate procedure types (e.g. two USER_DATA entries) are handled
		// correctly. A linear scan from the tracked index avoids the bug
		// where GetNextProcedure would always match the FIRST occurrence of
		// a repeated type and return the wrong successor.
		nextIdx := pCtx.CurrentProcIndex + 1
		if nextIdx < length {
			nextProcedure = p.Procedures[nextIdx]
			pCtx.CurrentProcIndex = nextIdx
		} else {
			p.Log.Infoln("no more procedures left")
		}
		return nextProcedure, nil
	}

	// check if custom Profile
	if len(p.Iterations) > 0 {
		pCtx.Log.Infoln("Current UE iteration", pCtx.CurrentItr)
		pCtx.Log.Infoln("Current UE procedure index", pCtx.CurrentProcIndex)
		itp, ok := p.PIterations[pCtx.CurrentItr]
		if !ok {
			return common.UNKNOWN_PROCEDURE, fmt.Errorf("unknown iteration %q", pCtx.CurrentItr)
		}
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
			return nextProcedure, nil
		}
		if pCtx.Repeat > 0 {
			pCtx.Repeat = pCtx.Repeat - 1
			pCtx.Log.Infoln("Repeat current iteration:", itp.Name, ", Repeat Count", pCtx.Repeat)
			pCtx.CurrentProcIndex = 1
			nextProcedure = itp.ProcMap[1]
			pCtx.Procedure = nextProcedure
			return nextProcedure, nil
		}
		pCtx.Log.Infoln("Iteration Complete", pCtx.CurrentItr)
		nextItr := itp.NextItr
		if nextItr != quit {
			nItr, ok := p.PIterations[nextItr]
			if !ok {
				return common.UNKNOWN_PROCEDURE, fmt.Errorf("unknown next iteration %q", nextItr)
			}
			pCtx.Log.Infoln("Going to next iteration", nItr.Name)
			pCtx.CurrentItr = nItr.Name
			pCtx.CurrentProcIndex = 1
			pCtx.Repeat = nItr.Repeat
			nextProcedure = nItr.ProcMap[1]
			pCtx.Procedure = nextProcedure
			return nextProcedure, nil
		}
	}
	pCtx.Log.Infoln("nothing more to execute for UE")
	return nextProcedure, nil
}
