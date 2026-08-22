// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package gnbamfworker

import (
	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/test"
	"github.com/omec-project/ngap/v2/ngapConvert"
	"github.com/omec-project/ngap/v2/ngapType"
	"github.com/omec-project/openapi/v2/models"
)

// HandleNGSetupResponse processes the NG Setup Response and updates GnbAmf
// context
func HandleNgSetupResponse(amf *gnbctx.GnbAmf, pdu *ngapType.NGAPPDU) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing NG Setup Response")
	var amfName *ngapType.AMFName
	var servedGUAMIList *ngapType.ServedGUAMIList
	var relativeAMFCapacity *ngapType.RelativeAMFCapacity
	var plmnSupportList *ngapType.PLMNSupportList
	// TODO Process optional IEs

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	successfulOutcome := pdu.SuccessfulOutcome
	if successfulOutcome == nil {
		amf.Log.Errorln("SuccessfulOutcome is nil")
		return
	}
	ngSetupResponse := successfulOutcome.Value.NGSetup
	if ngSetupResponse == nil {
		amf.Log.Errorln("NGSetupResponse is nil")
		return
	}

	amf.Log.Debugln("handle NG Setup response")
	for i := 0; i < len(ngSetupResponse.ProtocolIEs.List); i++ {
		ie := ngSetupResponse.ProtocolIEs.List[i]
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFName:
			amfName = ie.Value.AMFName
			amf.Log.Debugln("decode IE AMFName")
			if amfName == nil {
				amf.Log.Errorln("AMFName is nil")
				return
			}
		case ngapType.ProtocolIEIDServedGUAMIList:
			servedGUAMIList = ie.Value.ServedGUAMIList
			amf.Log.Debugln("decode IE ServedGUAMIList")
			if servedGUAMIList == nil {
				amf.Log.Errorln("ServedGUAMIList is nil")
				return
			}
		case ngapType.ProtocolIEIDRelativeAMFCapacity:
			relativeAMFCapacity = ie.Value.RelativeAMFCapacity
			amf.Log.Debugln("decode IE RelativeAMFCapacity")
			if relativeAMFCapacity == nil {
				amf.Log.Errorln("RelativeAMFCapacity is nil")
				return
			}
		case ngapType.ProtocolIEIDPLMNSupportList:
			plmnSupportList = ie.Value.PLMNSupportList
			amf.Log.Debugln("decode IE PLMNSupportList")
			if plmnSupportList == nil {
				amf.Log.Errorln("PLMNSupportList is nil")
				return
			}
		}
	}

	amf.SetAMFName(amfName.Value)
	amf.SetRelativeAMFCapacity(relativeAMFCapacity.Value)

	// Initializing the ServedGuamiList slice in GnbAmf if not already initialized
	// This will also clear any existing contents of ServedGuamiList within GnbAmf
	if len(amf.ServedGuamiList) != 0 || cap(amf.ServedGuamiList) == 0 {
		amf.ServedGuamiList = make([]models.Guami, 0, gnbctx.MaxNumOfServedGuamiList)
	}

	capOfGuamiList := cap(amf.ServedGuamiList)
	for _, servedGuamiItem := range servedGUAMIList.List {
		if len(amf.ServedGuamiList) >= capOfGuamiList {
			break
		}

		guamiSrc := servedGuamiItem.GUAMI

		// Parsing PLMNID into models.Guami
		plmnId, err := ngapConvert.PlmnIdToModels(guamiSrc.PLMNIdentity)
		if err != nil {
			amf.Log.Errorln("PlmnIdToModels returned:", err)
			return
		}

		guami := models.Guami{
			PlmnId: models.PlmnIdNid{
				Mcc: plmnId.GetMcc(),
				Mnc: plmnId.GetMnc(),
			},
			AmfId: ngapConvert.AmfIdToModels(guamiSrc.AMFRegionID.Value, guamiSrc.AMFSetID.Value, guamiSrc.AMFPointer.Value),
		}

		amf.ServedGuamiList = append(amf.ServedGuamiList, guami)
	}

	if len(amf.ServedGuamiList) == 0 {
		amf.Log.Errorln("NG Setup Response: Empty ServedGuamiList received")
	} /* else {
		// TODO: Need to check
	}*/

	// Initializing the PlmnSuportList slice in GnbAmf if not already initialized
	// This will also clear any existing contents of PlmnSupportList within GnbAmf
	if len(amf.PlmnSupportList) != 0 || cap(amf.PlmnSupportList) == 0 {
		amf.PlmnSupportList = make([]models.PlmnSnssai, 0, gnbctx.MaxNumOfPLMNs)
	}
	capOfPlmnSupportList := cap(amf.PlmnSupportList)
	for _, plmnSupportItem := range plmnSupportList.List {
		if len(amf.PlmnSupportList) >= capOfPlmnSupportList {
			break
		}

		// Parsing PLMNID into models.PlmnId
		plmnId, err := ngapConvert.PlmnIdToModels(plmnSupportItem.PLMNIdentity)
		if err != nil {
			amf.Log.Errorln("PlmnIdToModels returned:", err)
			return
		}

		// Parsing SNssaiList into models.Snssai
		snssaiList := make([]models.Snssai, 0, len(plmnSupportItem.SliceSupportList.List))
		for _, sliceSupportItem := range plmnSupportItem.SliceSupportList.List {
			snssai, err := ngapConvert.SNssaiToModels(sliceSupportItem.SNSSAI)
			if err != nil {
				amf.Log.Errorln("SNssaiToModels returned:", err)
				return
			}
			snssaiList = append(snssaiList, snssai)
		}
		plmnSI := models.PlmnSnssai{
			PlmnId:     plmnId,
			SNssaiList: snssaiList,
		}
		amf.PlmnSupportList = append(amf.PlmnSupportList, plmnSI)
	}

	if len(amf.PlmnSupportList) == 0 {
		amf.Log.Errorln("NG Setup Response: Empty PLMNSupportList received")
	} /*else {
		// TODO: Need to check whether it should be compared against some
		// existing list within gNodeB
	}*/

	amf.SetNgSetupStatus(true)
	amf.Log.Debugln("processed NG Setup Response")
}

func HandleNgSetupFailure(amf *gnbctx.GnbAmf, pdu *ngapType.NGAPPDU) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing NG Setup Failure")
	var cause *ngapType.Cause

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	UnSuccessfulOutcome := pdu.UnsuccessfulOutcome
	if UnSuccessfulOutcome == nil {
		amf.Log.Errorln("UnSuccessfulOutcome Message is nil")
		return
	}
	ngSetupFailure := UnSuccessfulOutcome.Value.NGSetup
	if ngSetupFailure == nil {
		amf.Log.Errorln("NGSetupFailure is nil")
		return
	}

	amf.Log.Debugln("handle NG Setup Failure")
	for i := 0; i < len(ngSetupFailure.ProtocolIEs.List); i++ {
		ie := ngSetupFailure.ProtocolIEs.List[i]
		if ie.Id.Value == ngapType.ProtocolIEIDCause {
			cause = ie.Value.Cause
			amf.Log.Debugln("decode IE Cause")
			if cause == nil {
				amf.Log.Errorln("Cause is nil")
				return
			}
			break
		}
		// TODO handle TimeToWait IE
	}

	test.PrintAndGetCause(cause)
	amf.SetNgSetupStatus(false)

	amf.Log.Debugln("processed NG Setup Failure")
}

func HandleDownlinkNasTransport(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Downlink Nas Transport")
	var gnbUeNgapId *ngapType.RANUENGAPID

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("Initiating Message is nil")
		return
	}
	downlinkNasTransport := initiatingMessage.Value.DownlinkNASTransport
	if downlinkNasTransport == nil {
		amf.Log.Errorln("DownlinkNASTransport is nil")
		return
	}

	amf.Log.Debugln("handle Downlink NAS Transport")
	for i := 0; i < len(downlinkNasTransport.ProtocolIEs.List); i++ {
		ie := downlinkNasTransport.ProtocolIEs.List[i]
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Debugln("decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				amf.Log.Errorln("RANUENGAPID is nil")
				return
			}
			break
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		amf.Log.Errorln("no GnbUe found corresponding to RANUENGAPID:", ngapId)
		return
	}

	SendToGnbUe(gnbue, common.DOWNLINK_NAS_TRANSPORT_EVENT, pdu, id)
}

func HandleInitialContextSetupRequest(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Initial Context Setup Request")
	var gnbUeNgapId *ngapType.RANUENGAPID

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("InitiatingMessage is nil")
		return
	}
	initialContextSetupRequest := initiatingMessage.Value.InitialContextSetup
	if initialContextSetupRequest == nil {
		amf.Log.Errorln("InitialContextSetupRequest is nil")
		return
	}

	amf.Log.Debugln("InitialContextSetupRequest")
	for _, ie := range initialContextSetupRequest.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Debugln("decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				amf.Log.Errorln("RANUENGAPID is nil")
				return
			}
			break
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		amf.Log.Errorln("no GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.INITIAL_CTX_SETUP_REQUEST_EVENT, pdu, id)
}

// TODO : Much of the code is repeated in each handler
func HandlePduSessResourceSetupRequest(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Pdu Session Resource Setup Request")
	var gnbUeNgapId *ngapType.RANUENGAPID

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("InitiatingMessage is nil")
		return
	}
	pduSessResourceSetupReq := initiatingMessage.Value.PDUSessionResourceSetup
	if pduSessResourceSetupReq == nil {
		amf.Log.Errorln("PDUSessionResourceSetupRequest is nil")
		return
	}

	for _, ie := range pduSessResourceSetupReq.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Debugln("decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				amf.Log.Errorln("RANUENGAPID is nil")
				return
			}
			break
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		amf.Log.Errorln("no GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT, pdu, id)
}

func HandlePduSessResourceReleaseCommand(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Pdu Session Resource Release Command")
	var gnbUeNgapId *ngapType.RANUENGAPID

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("InitiatingMessage is nil")
		return
	}
	pduSessResourceReleaseCmd := initiatingMessage.Value.PDUSessionResourceRelease
	if pduSessResourceReleaseCmd == nil {
		amf.Log.Errorln("PDUSessionResourceReleaseCommand is nil")
		return
	}

	for _, ie := range pduSessResourceReleaseCmd.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Debugln("decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				amf.Log.Errorln("RANUENGAPID is nil")
				return
			}
			break
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		amf.Log.Errorln("no GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.PDU_SESS_RESOURCE_RELEASE_COMMAND_EVENT, pdu, id)
}

// HandlePduSessResourceModifyRequest routes a PDU SESSION RESOURCE MODIFY REQUEST to the UE it
// concerns, in the same way the release command is routed.
func HandlePduSessResourceModifyRequest(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Pdu Session Resource Modify Request")

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("InitiatingMessage is nil")
		return
	}
	modifyRequest := initiatingMessage.Value.PDUSessionResourceModify
	if modifyRequest == nil {
		amf.Log.Errorln("PDUSessionResourceModifyRequest is nil")
		return
	}

	var gnbUeNgapId *ngapType.RANUENGAPID
	for _, ie := range modifyRequest.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			if gnbUeNgapId == nil {
				amf.Log.Errorln("RANUENGAPID is nil")
				return
			}
			break
		}
	}
	if gnbUeNgapId == nil {
		amf.Log.Errorln("RANUENGAPID IE is absent")
		return
	}

	gnbue := gnb.GnbUes.GetGnbCpUe(gnbUeNgapId.Value)
	if gnbue == nil {
		amf.Log.Errorln("no GnbUe found corresponding to RANUENGAPID:", gnbUeNgapId.Value)
		return
	}

	SendToGnbUe(gnbue, common.PDU_SESS_RESOURCE_MODIFY_REQUEST_EVENT, pdu, id)
}

func HandleUeCtxReleaseCommand(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}

	amf.Log.Debugln("processing Ue Context Release Command")

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}

	var ueNgapIds *ngapType.UENGAPIDs

	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("InitiatingMessage is nil")
		return
	}

	ueCtxRelCmd := initiatingMessage.Value.UEContextRelease
	if ueCtxRelCmd == nil {
		amf.Log.Errorln("UEContextReleaseCommand is nil")
		return
	}

	for _, ie := range ueCtxRelCmd.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDUENGAPIDs:
			ueNgapIds = ie.Value.UENGAPIDs
			if ueNgapIds == nil {
				amf.Log.Errorln("UENGAPIDs is nil")
				return
			}
		}
	}

	if ueNgapIds.Present == ngapType.UENGAPIDsPresentUENGAPIDPair {
		if ueNgapIds.UENGAPIDPair == nil {
			amf.Log.Errorln("UENGAPIDPair is nil")
			return
		}
	} else {
		/*TODO: Should add mapping for AMFUENGAPID vs GnbCpUeContext*/
		amf.Log.Errorln("no RANUENGAPID received")
		return
	}

	ngapId := ueNgapIds.UENGAPIDPair.RANUENGAPID.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		amf.Log.Errorln("no GnbUe found corresponding to RANUENGAPID:", ngapId)
		return
	}

	SendToGnbUe(gnbue, common.UE_CTX_RELEASE_COMMAND_EVENT, pdu, id)
}

// HandleHandoverRequest processes a HandoverRequest received by the target gNB from AMF.
// It routes the message to the next pre-registered target GnbCpUe from a FIFO queue
// (pre-registration is done by SimUe before triggering N2 handover on the source gNB).
func HandleHandoverRequest(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Handover Request")

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}

	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		amf.Log.Errorln("InitiatingMessage is nil")
		return
	}
	hoReq := initiatingMessage.Value.HandoverResourceAllocation
	if hoReq == nil {
		amf.Log.Errorln("HandoverRequest is nil")
		return
	}

	var amfUeNgapId *ngapType.AMFUENGAPID
	for _, ie := range hoReq.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDAMFUENGAPID {
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				amf.Log.Errorln("AMFUENGAPID is nil in HandoverRequest")
				return
			}
			break
		}
	}
	if amfUeNgapId == nil {
		amf.Log.Errorln("AMF UE NGAP ID not found in HandoverRequest")
		return
	}

	// The OMEC AMF allocates a fresh AMF-UE-NGAP-ID for the target-side RanUe
	// (see NewRanUe → AllocateAmfUeNgapID in the AMF), so we cannot key the
	// lookup by the source UE's AMF-UE-NGAP-ID.  A FIFO queue is used instead.
	gnbue := gnb.GnbUes.DequeueHandoverTarget()
	if gnbue == nil {
		amf.Log.Errorln("no pre-registered target GnbCpUe found for HandoverRequest")
		return
	}
	// Record the AMF-UE-NGAP-ID assigned for the target side.
	gnbue.AmfUeNgapId = amfUeNgapId.Value

	SendToGnbUe(gnbue, common.HO_REQUEST_EVENT, pdu, id)
}

// HandleHandoverCommand processes a HandoverCommand received by the source gNB from AMF.
// This is the SuccessfulOutcome of the HandoverPreparation procedure.
func HandleHandoverCommand(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Init()
		amf.Log.Errorln("amf is nil")
		return
	}
	amf.Log.Debugln("processing Handover Command")

	if pdu == nil {
		amf.Log.Errorln("NGAP Message is nil")
		return
	}
	if gnb == nil {
		amf.Log.Errorln("gNodeB context is nil")
		return
	}

	successfulOutcome := pdu.SuccessfulOutcome
	if successfulOutcome == nil {
		amf.Log.Errorln("SuccessfulOutcome is nil")
		return
	}
	hoCmd := successfulOutcome.Value.HandoverPreparation
	if hoCmd == nil {
		amf.Log.Errorln("HandoverCommand is nil")
		return
	}

	var gnbUeNgapId *ngapType.RANUENGAPID
	for _, ie := range hoCmd.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			if gnbUeNgapId == nil {
				amf.Log.Errorln("RANUENGAPID is nil in HandoverCommand")
				return
			}
			break
		}
	}
	if gnbUeNgapId == nil {
		amf.Log.Errorln("RAN UE NGAP ID not found in HandoverCommand")
		return
	}

	gnbue := gnb.GnbUes.GetGnbCpUe(gnbUeNgapId.Value)
	if gnbue == nil {
		amf.Log.Errorln("no GnbCpUe found for RANUENGAPID:", gnbUeNgapId.Value)
		return
	}

	SendToGnbUe(gnbue, common.HO_COMMAND_EVENT, pdu, id)
}
