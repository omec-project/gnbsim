// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbcpueworker

import (
	"encoding/binary"
	"fmt"
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"gnbsim/gnodeb/worker/gnbupfworker"
	"gnbsim/gnodeb/worker/gnbupueworker"
	"gnbsim/util/ngapTestpacket"
	"gnbsim/util/test"

	"github.com/free5gc/aper"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
)

func HandleConnectRequest(gnbue *context.GnbCpUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling Connection Request Event from Ue")
	gnbue.Supi = msg.Supi
	gnbue.WriteUeChan = msg.CommChan
}

func HandleInitialUEMessage(gnbue *context.GnbCpUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling Initial UE Event")

	sendMsg, err := test.GetInitialUEMessage(gnbue.GnbUeNgapId, msg.NasPdus[0], "")
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

func HandleDownlinkNasTransport(gnbue *context.GnbCpUe, msg *common.N2Message) {
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
	var pdus common.NasPduList
	pdus = append(pdus, nasPdu.Value)
	SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, pdus)
	gnbue.Log.Traceln("Sent DL Information Transfer Event to UE")
}

func HandleUlInfoTransfer(gnbue *context.GnbCpUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling UL Information Transfer Event")

	gnbue.Log.Traceln("Creating Uplink NAS Transport Message")
	sendMsg, err := test.GetUplinkNASTransport(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, msg.NasPdus[0])
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

func HandleInitialContextSetupRequest(gnbue *context.GnbCpUe, msg *common.N2Message) {
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

	var pdus common.NasPduList
	pdus = append(pdus, nasPdu.Value)
	SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, pdus)
	gnbue.Log.Traceln("Sent DL Information Transfer Event to UE")
}

// TODO: Error handling
func HandlePduSessResourceSetupRequest(gnbue *context.GnbCpUe, msg *common.N2Message) {
	gnbue.Log.Traceln("Handling PDU Session Resource Setup Request Message")

	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdus common.NasPduList
	var pduSessResourceSetupReqList *ngapType.PDUSessionResourceSetupListSUReq

	pdu := msg.NgapPdu
	if pdu == nil {
		gnbue.Log.Errorln("NgapPdu is nil")
		return
	}

	initiatingMessage := pdu.InitiatingMessage
	pduSessResourceSetupReq := initiatingMessage.Value.PDUSessionResourceSetupRequest

	for _, ie := range pduSessResourceSetupReq.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				gnbue.Log.Errorln("AMFUENGAPID is nil")
				return
			}
		case ngapType.ProtocolIEIDPDUSessionResourceSetupListSUReq:
			pduSessResourceSetupReqList = ie.Value.PDUSessionResourceSetupListSUReq
			if pduSessResourceSetupReqList == nil || len(pduSessResourceSetupReqList.List) == 0 {
				gnbue.Log.Errorln("PDUSessionResourceSetupListSUReq is empty")
				return
			}
		}
	}

	//var pduSessions []ngapTestpacket.PduSession
	var upDataSet []*common.UserPlaneData

	// supporting only one pdu session currently
	for _, item := range pduSessResourceSetupReqList.List[:1] {

		resourceSetupRequestTransfer := ngapType.PDUSessionResourceSetupRequestTransfer{}
		err := aper.UnmarshalWithParams(item.PDUSessionResourceSetupRequestTransfer,
			&resourceSetupRequestTransfer, "valueExt")
		if err != nil {
			gnbue.Log.Errorln("UnmarshalWithParams returned:", err)
			return
		}

		var gtpTunnel *ngapType.GTPTunnel
		var pduSessType *ngapType.PDUSessionType
		var qosFlowSetupReqList *ngapType.QosFlowSetupRequestList
		for _, ie := range resourceSetupRequestTransfer.ProtocolIEs.List {
			switch ie.Id.Value {
			case ngapType.ProtocolIEIDULNGUUPTNLInformation:
				gtpTunnel = ie.Value.ULNGUUPTNLInformation.GTPTunnel
				if gtpTunnel == nil {
					gnbue.Log.Errorln("GTPTunnel is nil")
					return
				}
			case ngapType.ProtocolIEIDPDUSessionType:
				pduSessType = ie.Value.PDUSessionType
				if pduSessType == nil {
					gnbue.Log.Errorln("PDUSessionType is nil")
					return
				}
			case ngapType.ProtocolIEIDQosFlowSetupRequestList:
				qosFlowSetupReqList = ie.Value.QosFlowSetupRequestList
				if qosFlowSetupReqList == nil || len(qosFlowSetupReqList.List) == 0 {
					gnbue.Log.Errorln("QosFlowSetupRequestList is empty")
					return
				}
			}
		}

		ulteid := binary.BigEndian.Uint32(gtpTunnel.GTPTEID.Value)
		dlteid, err := gnbue.Gnb.DlTeidGenerator.Allocate()
		if err != nil {
			gnbue.Log.Errorln("ID Generator Allocate() returned:", err)
			return
		}
		upfIp, _ := ngapConvert.IPAddressToString(gtpTunnel.TransportLayerAddress)

		gnbupue := context.NewGnbUpUe(uint32(dlteid), ulteid, gnbue.Gnb)
		gnbupue.Snssai = ngapConvert.SNssaiToModels(item.SNSSAI)
		gnbupue.PduSessId = item.PDUSessionID.Value
		gnbupue.PduSessType = test.PDUSessionTypeToModels(*pduSessType)
		pduSess := &ngapTestpacket.PduSession{}
		pduSess.PduSessId = gnbupue.PduSessId

		gnbue.Log.Infoln("PDU Session ID:", gnbupue.PduSessId)
		gnbue.Log.Infoln("S-NSSAI - SST: ", gnbupue.Snssai.Sst)
		gnbue.Log.Infoln("S-NSSAI - SD: ", gnbupue.Snssai.Sd)
		gnbue.Log.Infoln("UL GTP-TEID: ", ulteid)
		gnbue.Log.Infoln("DL GTP-TEID: ", dlteid)
		gnbue.Log.Infoln("UPF Endpoint IP: ", upfIp)
		gnbue.Log.Infoln("PDU Session Type: ", gnbupue.PduSessType)

		var qosFlowId int64
		var qosChar ngapType.QosCharacteristics
		var arp ngapType.AllocationAndRetentionPriority
		var nonDynamic5QI *ngapType.NonDynamic5QIDescriptor
		for _, qosFlowSetupReqItem := range qosFlowSetupReqList.List {
			qosFlowId = qosFlowSetupReqItem.QosFlowIdentifier.Value
			qosChar = qosFlowSetupReqItem.QosFlowLevelQosParameters.QosCharacteristics
			arp = qosFlowSetupReqItem.QosFlowLevelQosParameters.AllocationAndRetentionPriority

			gnbue.Log.Infoln("QoS Flow Id:", qosFlowId)
			if qosChar.Present == ngapType.QosCharacteristicsPresentNonDynamic5QI {
				nonDynamic5QI = qosChar.NonDynamic5QI
				if nonDynamic5QI == nil {
					gnbue.Log.Errorln("NonDynamic5QI is nil")
					return
				}
				gnbue.Log.Infoln("Non Dynamic 5QI:", nonDynamic5QI.FiveQI.Value)
			}
			gnbue.Log.Infoln("ARP Priority Level:", arp.PriorityLevelARP.Value)
			gnbue.Log.Infoln("Pre-emption Capability:", arp.PreEmptionCapability.Value)
			gnbue.Log.Infoln("Pre-emption Vulnerability:", arp.PreEmptionVulnerability.Value)

			pduSess.SuccessQfiList = append(pduSess.SuccessQfiList, qosFlowId)
			gnbupue.AddQosFlow(qosFlowId, &qosFlowSetupReqItem)
		}

		if item.PDUSessionNASPDU != nil {
			nasPdus = append(nasPdus, item.PDUSessionNASPDU.Value)
		}

		gnbupf, created := gnbue.Gnb.GnbPeers.GetOrAddGnbUpf(upfIp)
		if created {
			go gnbupfworker.Init(gnbupf)
		}
		gnbupue.Upf = gnbupf
		gnbue.AddGnbUpUe(gnbupue.PduSessId, gnbupue)

		//pduSessions = append(pduSessions, pduSess)
		upData := &common.UserPlaneData{}
		upData.CommChan = gnbupue.ReadUlChan
		upData.PduSess = pduSess
		upDataSet = append(upDataSet, upData)

		// Create a Data Bearer Setup Request populated with pdu sessions
		// and send it to Sim UE, upon receiving all the Data Bearer Setup
		// Response, Create PDU Session Resource Setup Response with successful
		// pdu session ids and failed pdu session ids and send it to amf
		// also add real ue data channels to gnbupue and associate the gnbupue
		// with the teid in gnbupf-maps

	}

	SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nasPdus)
	gnbue.Log.Traceln("Sent DL Information Transfer Event to UE")

	uemsg := common.UuMessage{}
	uemsg.Event = common.DATA_BEARER_SETUP_REQUEST_EVENT
	uemsg.Interface = common.UU_INTERFACE
	uemsg.UPData = upDataSet
	gnbue.WriteUeChan <- &uemsg
	gnbue.Log.Infoln("Sent Data Radio Bearer Setup Event to Ue")
}

func HandleDataBearerSetupResponse(gnbue *context.GnbCpUe, msg *common.UuMessage) {
	gnbue.Log.Traceln("Handling Initial UE Event")

	var pduSessions []*ngapTestpacket.PduSession
	for _, item := range msg.UPData {
		pduSess := item.PduSess
		if !pduSess.Success {
			gnbue.RemoveGnbUpUe(pduSess.PduSessId)
		} else {
			gnbUpUe := gnbue.GetGnbUpUe(pduSess.PduSessId)
			// TODO: Addition to this map should only be through GnbUpfWorker
			// routine. This will help in replacing sync map with normal map
			// Thus will help avoid lock unlock operation on per downlink message
			gnbUpUe.Upf.GnbUpUes.AddGnbUpUe(gnbUpUe.DlTeid, true, gnbUpUe)
			gnbUpUe.WriteUeChan = item.CommChan
			go gnbupueworker.Init(gnbUpUe)
		}
		pduSessions = append(pduSessions, pduSess)
	}

	ngapPdu, err := test.GetPDUSessionResourceSetupResponse(pduSessions,
		gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, gnbue.Gnb.GnbN3Ip)
	if err != nil {
		fmt.Println("Failed to create - NGAP-PDU Session Resource Setup Response")
		return
	}

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, ngapPdu)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}
	gnbue.Log.Traceln("Sent PDU Session Resource Setup Response Message to AMF")
}
