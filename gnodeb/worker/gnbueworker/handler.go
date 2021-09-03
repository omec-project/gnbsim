// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbueworker

import (
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"gnbsim/util/test"

	"github.com/free5gc/ngap/ngapType"
)

func HandleConnectRequest(gnbue *context.GnbUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling Connection Request Event from Ue")
	gnbue.Supi = msg.Supi
	gnbue.WriteUeChan = msg.UeChan
}

func HandleInitialUEMessage(gnbue *context.GnbUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling Initial UE Event")

	sendMsg, err := test.GetInitialUEMessage(gnbue.GnbUeNgapId, msg.NasPdu, "")
	if err != nil {
		gnbue.Log.Errorln("GetInitialUEMessage failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}

	gnbue.Log.Traceln("Sent Initial UE Message to AMF")
}

func HandleDownlinkNasTransport(gnbue *context.GnbUe, msg *common.N2Message) {
	gnbue.Log.Traceln("Handling Downlink NAS Transport Message")

	// Need not perform other checks as they are validated at gnbamfworker level
	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdu *ngapType.NASPDU

	pdu := msg.NgapPdu
	if pdu == nil {
		gnbue.Log.Errorln("NgapPdu is nil")
		return
	}

	// Null checks are already performed at gnbamfworker level
	initiatingMessage := pdu.InitiatingMessage
	downlinkNasTransport := initiatingMessage.Value.DownlinkNASTransport

	for i := 0; i < len(downlinkNasTransport.ProtocolIEs.List); i++ {
		ie := downlinkNasTransport.ProtocolIEs.List[i]
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				gnbue.Log.Errorln("AMFUENGAPID is nil")
				return
			}
		case ngapType.ProtocolIEIDNASPDU:
			nasPdu = ie.Value.NASPDU
			if nasPdu == nil {
				gnbue.Log.Errorln("NASPDU is nil")
				return
			}
		}
	}

	//TODO: check what needs to be done with AmfUeNgapId on every DownlinkNasTransport message
	gnbue.AmfUeNgapId = amfUeNgapId.Value
	SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nasPdu.Value)
	gnbue.Log.Traceln("Sent DL Information Transfer Event to UE")
}

func HandleUlInfoTransfer(gnbue *context.GnbUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling UL Information Transfer Event")

	gnbue.Log.Traceln("Creating Uplink NAS Transport Message")
	sendMsg, err := test.GetUplinkNASTransport(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, msg.NasPdu)
	if err != nil {
		gnbue.Log.Errorln("GetUplinkNASTransport failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}

	gnbue.Log.Traceln("Sent Uplink NAS Transport Message to AMF")
}

func HandleInitialContextSetupRequest(gnbue *context.GnbUe, msg *common.N2Message) {
	gnbue.Log.Traceln("Handling Initial Context Setup Request Message")

	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdu *ngapType.NASPDU

	pdu := msg.NgapPdu
	if pdu == nil {
		gnbue.Log.Errorln("NgapPdu is nil")
		return
	}

	// Null checks are already performed at gnbamfworker level
	initiatingMessage := pdu.InitiatingMessage
	initialContextSetupRequest := initiatingMessage.Value.InitialContextSetupRequest

	for _, ie := range initialContextSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				gnbue.Log.Errorln("AMFUENGAPID is nil")
				return
			}
		case ngapType.ProtocolIEIDNASPDU:
			nasPdu = ie.Value.NASPDU
			if nasPdu == nil {
				gnbue.Log.Errorln("NASPDU is nil")
				return
			}
		}
	}

	//TODO: Handle other mandatory IEs
	resp, err := test.GetInitialContextSetupResponse(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId)
	if err != nil {
		gnbue.Log.Errorln("Failed to create Initial Context Setup Response Message ")
		return
	}

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, resp)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}
	gnbue.Log.Traceln("Sent Initial Context Setup Response Message to UE")

	SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nasPdu.Value)
	gnbue.Log.Traceln("Sent DL Information Transfer Event to UE")
}
