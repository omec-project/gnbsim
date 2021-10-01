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
	GetInterfaceType() InterfaceType
}

type DefaultMessage struct {
	Event     EventType
	Interface InterfaceType
}

func (msg *DefaultMessage) GetEventType() EventType {
	return msg.Event
}

func (msg *DefaultMessage) GetInterfaceType() InterfaceType {
	return msg.Interface
}

// N2Message is used to transfer information gnodeb the GNodeB components
type N2Message struct {
	DefaultMessage
	NgapPdu *ngapType.NGAPPDU
}

type NasPduList [][]byte

// UuMessage is used to transfer information between the UE and GNodeB
type UuMessage struct {
	DefaultMessage
	Supi string
	// Encoded NAS message
	NasPdus NasPduList
	Extras  EventData
	UPData  []*UserPlaneData
	// channel that a src entity can optionally send to the target entity.
	// Target entity will use this channel to write to the src entity
	CommChan chan InterfaceMessage
}

// UuMessage is used to transfer information between the UE and GNodeB
type ProfileMessage struct {
	DefaultMessage
	Supi     string
	Proc     ProcedureType
	ErrorMsg error
}

func (msg *ProfileMessage) GetEventType() EventType {
	return msg.Event
}

func (msg *ProfileMessage) GetInterfaceType() InterfaceType {
	return msg.Interface
}

type EventData struct {
	Cause uint8

	/* Decoded NAS message */
	NasMsg *nas.Message
}

type UserPlaneData struct {
	PduSess *ngapTestpacket.PduSession

	/* Channel to be used to by target entity to send data packets for this
	   pdu session */
	CommChan chan InterfaceMessage
}
