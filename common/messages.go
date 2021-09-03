// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package common

import (
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

// UuMessage is used to transfer information between the UE and GNodeB
type UuMessage struct {
	DefaultMessage
	Supi string
	// Encoded NAS message
	NasPdu []byte
	Extras EventData
	// Channel to communicate with UE
	UeChan chan InterfaceMessage
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
	// Decoded NAS message
	NasMsg *nas.Message
}
