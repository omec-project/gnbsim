// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbueworker

import (
	"gnbsim/gnodeb/context"
	intfc "gnbsim/interfacecommon"
	"gnbsim/util/test"
	"log"

	"github.com/free5gc/ngap/ngapType"
)

func HandleUeConnection(gnbue *context.GnbUe, msg *intfc.UuMessage) {
	gnbue.Supi = msg.Supi
	gnbue.WriteUeChan = msg.UeChan
}

func HandleInitialUEMessage(gnbue *context.GnbUe, msg *intfc.UuMessage) {
	sendMsg, err := test.GetInitialUEMessage(gnbue.GnbUeNgapId, msg.NasPdu, "")
	if err != nil {
		log.Println("Error: GetInitialUEMessage failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg)
	if err != nil {
		log.Println("Error: SendToPeer failed:", err)
		return
	}

	log.Println("Sent InitialUEMessage")
}

func HandleDownlinkNasTransport(gnbue *context.GnbUe, msg *intfc.N2Message) {
	// Need not perform other checks as they are validated at gnbamfworker level
	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdu *ngapType.NASPDU

	pdu := msg.NgapPdu
	if pdu == nil {
		log.Println("Error: NgapPdu is nil")
		return
	}

	// Null checks are already performed at gnbamfworker level
	initiatingMessage := pdu.InitiatingMessage
	downlinkNasTransport := initiatingMessage.Value.DownlinkNASTransport

	log.Println("Handle Downlink NAS Transport")
	for i := 0; i < len(downlinkNasTransport.ProtocolIEs.List); i++ {
		ie := downlinkNasTransport.ProtocolIEs.List[i]
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDRANUENGAPID:
			log.Println("Decode IE AMFUENGAPID")
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				log.Println("AMFUENGAPID is nil")
				return
			}
		case ngapType.ProtocolIEIDNASPDU:
			log.Println("Decode IE NASPDU")
			nasPdu = ie.Value.NASPDU
			if nasPdu == nil {
				log.Println("NASPDU is nil")
				return
			}
		}
	}

	//TODO: check what needs to be done with AmfUeNgapId on every DownlinkNasTransport message
	gnbue.AmfUeNgapId = amfUeNgapId.Value
	uemsg := intfc.UuMessage{}
	uemsg.Event = intfc.GNB_DOWNLINK_NAS_TRANSPORT
	uemsg.NasPdu = nasPdu.Value
	gnbue.WriteUeChan <- &uemsg
}

func HandleUplinkNasTransport(gnbue *context.GnbUe, msg *intfc.UuMessage) {
	sendMsg, err := test.GetUplinkNASTransport(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, msg.NasPdu)
	if err != nil {
		log.Println("Error: GetUplinkNASTransport failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg)
	if err != nil {
		log.Println("Error: SendToPeer failed:", err)
		return
	}

	log.Println("Sent UplinkNASTransport Message")
}

func HandleInitialContextSetupRequest(gnbue *context.GnbUe, msg *intfc.N2Message) {
	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdu *ngapType.NASPDU

	pdu := msg.NgapPdu
	if pdu == nil {
		log.Println("Error: NgapPdu is nil")
		return
	}

	// Null checks are already performed at gnbamfworker level
	initiatingMessage := pdu.InitiatingMessage
	initialContextSetupRequest := initiatingMessage.Value.InitialContextSetupRequest

	log.Println("Handle Downlink NAS Transport")
	for _, ie := range initialContextSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDRANUENGAPID:
			log.Println("Decode IE AMFUENGAPID")
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				log.Println("AMFUENGAPID is nil")
				return
			}
		case ngapType.ProtocolIEIDNASPDU:
			log.Println("Decode IE NASPDU")
			nasPdu = ie.Value.NASPDU
			if nasPdu == nil {
				log.Println("NASPDU is nil")
				return
			}
		}
	}

	//TODO: Handle other mandatory IEs
	resp, err := test.GetInitialContextSetupResponse(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId)
	if err != nil {
		log.Println("Failed to get - Initial Context Setup Response Message ")
		return
	}

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, resp)
	if err != nil {
		log.Println("Error: SendToPeer failed:", err)
		return
	}

	log.Println("Sent InitialUEMessage")

	uemsg := intfc.UuMessage{}
	uemsg.Event = intfc.GNB_DOWNLINK_NAS_TRANSPORT
	uemsg.NasPdu = nasPdu.Value
	gnbue.WriteUeChan <- &uemsg
}
