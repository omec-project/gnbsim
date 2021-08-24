// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package interfacecommon

import "github.com/free5gc/ngap/ngapType"

// TODO should be moved out to different package so as to be used by other packages
type InterfaceMessage interface {
	GetEventType() EventType
	GetInterfaceType() InterfaceType
}

// GnbMessage is used to transfer information gnodeb the GNodeB components
type N2Message struct {
	Event     EventType
	Interface InterfaceType
	NgapPdu   *ngapType.NGAPPDU
}

// TODO : Try moving Event and Interface type to a seperate struct and embed
// it in AmfMessage struct. This way we need not repeat this two fields and
// methods for each concrete InterfaceMessage

func (msg *N2Message) GetEventType() EventType {
	return msg.Event
}

func (msg *N2Message) GetInterfaceType() InterfaceType {
	return msg.Interface
}

// Move out to common package
// UeMessage is used to transfer information between the UE and GNodeB
type UuMessage struct {
	Event     EventType
	Interface InterfaceType
	Supi      string
	NasPdu    []byte
	//channel to communicate with UE
	UeChan chan<- *UuMessage
}

func (msg *UuMessage) GetEventType() EventType {
	return msg.Event
}

func (msg *UuMessage) GetInterfaceType() InterfaceType {
	return msg.Interface
}
