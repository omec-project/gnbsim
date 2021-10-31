// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package common

import (
	"gnbsim/util/ngapTestpacket"

	"github.com/free5gc/ngap/ngapType"
	"github.com/omec-project/nas"
)

type InterfaceMessage interface {
	GetEventType() EventType
	GetErrorMsg() error
}

type DefaultMessage struct {
	Event EventType

	// Any error associated with this message
	Error error
}

func (msg *DefaultMessage) GetEventType() EventType {
	return msg.Event
}

func (msg *DefaultMessage) GetErrorMsg() error {
	return msg.Error
}

// Message received over N2 interface
type N2Message struct {
	DefaultMessage
	NgapPdu *ngapType.NGAPPDU
}

type NasPduList [][]byte

// UuMessage is used to carry information between the UE and GNodeB
type UuMessage struct {
	DefaultMessage
	Supi string

	// Encoded NAS message
	NasPdus  NasPduList
	DBParams []*DataBearerParams

	// channel that a src entity can optionally send to the target entity.
	// Target entity will use this channel to write to the src entity
	CommChan chan InterfaceMessage
}

// ProfileMessage is used to carry information between the Profile and SimUe
type ProfileMessage struct {
	DefaultMessage
	Supi string
	Proc ProcedureType
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
	Payload []byte
	Qfi     int64
}

// TransportMessage is used to carry raw message received over the transport
// layer
type TransportMessage struct {
	DefaultMessage
	RawPkt []byte
}

// UeMessage is used to carry information between SimUe and RealUe
type UeMessage struct {
	DefaultMessage

	// Decoded NAS message
	NasMsg *nas.Message

	// Number of user data packets to be generated as directed by profile
	UserDataPktCount int
}
