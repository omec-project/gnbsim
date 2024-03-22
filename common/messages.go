// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
	"github.com/omec-project/gnbsim/util/test"
	"github.com/omec-project/nas"
	"github.com/omec-project/ngap/ngapType"
)

type InterfaceMessage interface {
	GetEventType() EventType
	GetErrorMsg() error
}

type DefaultMessage struct {
	// Any error associated with this message
	Error error

	Event EventType
}

func (msg *DefaultMessage) GetEventType() EventType {
	return msg.Event
}

func (msg *DefaultMessage) GetErrorMsg() error {
	return msg.Error
}

// Message received over N2 interface
type N2Message struct {
	NgapPdu *ngapType.NGAPPDU
	DefaultMessage
	Id uint64
}

type NasPduList [][]byte

// UuMessage is used to carry information between the UE and GNodeB
type UuMessage struct {
	DefaultMessage
	Supi string

	// meta data
	Tmsi string

	// channel that a src entity can optionally send to the target entity.
	// Target entity will use this channel to write to the src entity
	CommChan chan InterfaceMessage

	// Encoded NAS message
	NasPdus  NasPduList
	DBParams []*DataBearerParams

	/* Real UE simply resends this value in the response message to gNB
	   While setting up Data Bearers, this helps gNB in understanding the
	   triggering procedure.
	*/
	TriggeringEvent EventType

	// Unique Message Id
	Id uint64
}

// ProfileMessage is used to carry information between the Profile and SimUe
type ProfileMessage struct {
	DefaultMessage
	Supi string
	Proc ProcedureType
}

// SummaryMessage is used to carry profile execution summary. Sent by profile
// routines to main routine
type SummaryMessage struct {
	DefaultMessage
	ProfileType   string
	ProfileName   string
	ErrorList     []error
	UePassedCount uint
	UeFailedCount uint
}

// DataBearerParams hold information require to setup data bearer(path) between
// RealUe and gNB
type DataBearerParams struct {
	PduSess *ngapTestpacket.PduSession

	// Channel to be used by target entity to send data packets for this pdu
	// session
	CommChan chan InterfaceMessage
}

// UserDataMessage is used to carry user data between Real UE and gNodeB
type UserDataMessage struct {
	DefaultMessage
	Qfi     *uint8
	Payload []byte
}

type N3Message struct {
	Pdu *test.GtpPdu
	DefaultMessage
}

// TransportMessage is used to carry raw message received over the transport
// layer
type TransportMessage struct {
	DefaultMessage
	RawPkt []byte
}

// UeMessage is used to carry information within UE
type UeMessage struct {
	DefaultMessage

	// Decoded NAS message
	NasMsg *nas.Message

	CommChan chan InterfaceMessage

	// default destination of data pkt
	DefaultAs string

	// Number of user data packets to be generated as directed by profile
	UserDataPktCount int

	// User data packets generating interval as directed by profile
	UserDataPktInterval int

	// Unique Message Id
	Id uint64
}
