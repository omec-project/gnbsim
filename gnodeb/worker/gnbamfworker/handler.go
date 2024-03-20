// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package gnbamfworker

import (
	amfctx "github.com/omec-project/amf/context"
	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/test"
	"github.com/omec-project/ngap/ngapConvert"
	"github.com/omec-project/ngap/ngapType"
	"github.com/omec-project/openapi/models"
)

// HandleNGSetupResponse processes the NG Setup Response and updates GnbAmf
// context
func HandleNgSetupResponse(amf *gnbctx.GnbAmf, pdu *ngapType.NGAPPDU) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Log.Errorln("ran is nil")
		return
	}
	amf.Log.Traceln("Processing NG Setup Response")
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
		amf.Log.Errorln("Initiating Message is nil")
		return
	}
	ngSetupResponse := successfulOutcome.Value.NGSetupResponse
	if ngSetupResponse == nil {
		amf.Log.Errorln("NGSetupResponse is nil")
		return
	}

	amf.Log.Traceln("Handle NG Setup response")
	for i := 0; i < len(ngSetupResponse.ProtocolIEs.List); i++ {
		ie := ngSetupResponse.ProtocolIEs.List[i]
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFName:
			amfName = ie.Value.AMFName
			amf.Log.Traceln("Decode IE AMFName")
			if amfName == nil {
				amf.Log.Errorln("AMFName is nil")
				return
			}
		case ngapType.ProtocolIEIDServedGUAMIList:
			servedGUAMIList = ie.Value.ServedGUAMIList
			amf.Log.Traceln("Decode IE ServedGUAMIList")
			if servedGUAMIList == nil {
				amf.Log.Errorln("ServedGUAMIList is nil")
				return
			}
		case ngapType.ProtocolIEIDRelativeAMFCapacity:
			relativeAMFCapacity = ie.Value.RelativeAMFCapacity
			amf.Log.Traceln("Decode IE RelativeAMFCapacity")
			if relativeAMFCapacity == nil {
				amf.Log.Errorln("RelativeAMFCapacity is nil")
				return
			}
		case ngapType.ProtocolIEIDPLMNSupportList:
			plmnSupportList = ie.Value.PLMNSupportList
			amf.Log.Traceln("Decode IE PLMNSupportList")
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
		amf.ServedGuamiList = gnbctx.NewServedGUAMIList()
	}

	capOfGuamiList := cap(amf.ServedGuamiList)
	for i := 0; i < len(servedGUAMIList.List); i++ {
		servedGuamiItem := servedGUAMIList.List[i]
		guamiSrc := servedGuamiItem.GUAMI
		var guami models.Guami

		// Parsing PLMNID into models.Guami
		plmnId := ngapConvert.PlmnIdToModels(guamiSrc.PLMNIdentity)
		guami.PlmnId = &plmnId

		// Parsing AMF Region, Set and Pointer to models.Guami
		amfRegId := guamiSrc.AMFRegionID.Value
		amfSetId := guamiSrc.AMFSetID.Value
		amfPtr := guamiSrc.AMFPointer.Value
		guami.AmfId = ngapConvert.AmfIdToModels(amfRegId, amfSetId, amfPtr)

		if len(amf.ServedGuamiList) < capOfGuamiList {
			amf.ServedGuamiList = append(amf.ServedGuamiList, guami)
		} else {
			break
		}
	}

	if len(amf.ServedGuamiList) == 0 {
		amf.Log.Errorln("NG Setup Response : Empty ServedGuamiList received")
	} /* else {
		// TODO: Need to check
	}*/

	// Initializing the PlmnSuportList slice in GnbAmf if not already initialized
	// This will also clear any existing contents of PlmnSupportList within GnbAmf
	if len(amf.PlmnSupportList) != 0 || cap(amf.PlmnSupportList) == 0 {
		amf.PlmnSupportList = gnbctx.NewPlmnSupportList()
	}
	capOfPlmnSupportList := cap(amf.PlmnSupportList)
	for _, plmnSupportItem := range plmnSupportList.List {
		plmnSI := amfctx.NewPlmnSupportItem()

		// Parsing PLMNID into models.Guami
		plmnSI.PlmnId = ngapConvert.PlmnIdToModels(plmnSupportItem.PLMNIdentity)

		// Parsing SNssaiList into models.Snssai
		capOfSNssaiList := cap(plmnSI.SNssaiList)
		for _, sliceSupportItem := range plmnSupportItem.SliceSupportList.List {
			if len(plmnSI.SNssaiList) < capOfSNssaiList {
				plmnSI.SNssaiList = append(plmnSI.SNssaiList, ngapConvert.SNssaiToModels(sliceSupportItem.SNSSAI))
			} else {
				break
			}
		}
		if len(amf.PlmnSupportList) < capOfPlmnSupportList {
			amf.PlmnSupportList = append(amf.PlmnSupportList, plmnSI)
		} else {
			break
		}
	}

	if len(amf.PlmnSupportList) == 0 {
		amf.Log.Errorln("NG Setup Response : Empty PLMNSupportList received")
	} /*else {
		// TODO: Need to check whether it should be compared against some
		// existing list within gNodeB
	}*/

	amf.SetNgSetupStatus(true)
	amf.Log.Traceln("Processed NG Setup Response")
}

func HandleNgSetupFailure(amf *gnbctx.GnbAmf, pdu *ngapType.NGAPPDU) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Log.Errorln("ran is nil")
		return
	}
	amf.Log.Traceln("Processing NG Setup Failure")
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
	ngSetupFailure := UnSuccessfulOutcome.Value.NGSetupFailure
	if ngSetupFailure == nil {
		amf.Log.Errorln("NGSetupResponse is nil")
		return
	}

	amf.Log.Traceln("Handle NG Setup Failure")
	for i := 0; i < len(ngSetupFailure.ProtocolIEs.List); i++ {
		ie := ngSetupFailure.ProtocolIEs.List[i]
		if ie.Id.Value == ngapType.ProtocolIEIDCause {
			cause = ie.Value.Cause
			amf.Log.Traceln("Decode IE Cause")
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

	amf.Log.Traceln("Processed NG Setup Failure")
}

func HandleDownlinkNasTransport(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Log.Errorln("ran is nil")
		return
	}
	amf.Log.Traceln("Processing Downlink Nas Transport")
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

	amf.Log.Traceln("Handle Downlink NAS Transport")
	for i := 0; i < len(downlinkNasTransport.ProtocolIEs.List); i++ {
		ie := downlinkNasTransport.ProtocolIEs.List[i]
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Traceln("Decode IE RANUENGAPID")
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
		amf.Log.Errorln("No GnbUe found corresponding to RANUENGAPID:", ngapId)
		return
	}

	SendToGnbUe(gnbue, common.DOWNLINK_NAS_TRANSPORT_EVENT, pdu, id)
}

func HandleInitialContextSetupRequest(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Log.Errorln("ran is nil")
		return
	}
	amf.Log.Traceln("Processing Initial Context Setup Request")
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
	initialContextSetupRequest := initiatingMessage.Value.InitialContextSetupRequest
	if initialContextSetupRequest == nil {
		amf.Log.Errorln("InitialContextSetupRequest is nil")
		return
	}

	amf.Log.Traceln("InitialContextSetupRequest")
	for _, ie := range initialContextSetupRequest.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Traceln("Decode IE RANUENGAPID")
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
		amf.Log.Errorln("No GnbUe found corresponding to RANUENGAPID:")
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
		amf.Log.Errorln("ran is nil")
		return
	}
	amf.Log.Traceln("Processing Pdu Session Resource Setup Request")
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
	pduSessResourceSetupReq := initiatingMessage.Value.PDUSessionResourceSetupRequest
	if pduSessResourceSetupReq == nil {
		amf.Log.Errorln("PDUSessionResourceSetupRequest is nil")
		return
	}

	for _, ie := range pduSessResourceSetupReq.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Traceln("Decode IE RANUENGAPID")
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
		amf.Log.Errorln("No GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT, pdu, id)
}

func HandlePduSessResourceReleaseCommand(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Log.Errorln("ran is nil")
		return
	}
	amf.Log.Traceln("Processing Pdu Session Resource Release Command")
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
	pduSessResourceReleaseCmd := initiatingMessage.Value.PDUSessionResourceReleaseCommand
	if pduSessResourceReleaseCmd == nil {
		amf.Log.Errorln("PDUSessionResourceReleaseCommand is nil")
		return
	}

	for _, ie := range pduSessResourceReleaseCmd.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			amf.Log.Traceln("Decode IE RANUENGAPID")
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
		amf.Log.Errorln("No GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.PDU_SESS_RESOURCE_RELEASE_COMMAND_EVENT, pdu, id)
}

func HandleUeCtxReleaseCommand(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf,
	pdu *ngapType.NGAPPDU, id uint64,
) {
	if amf == nil {
		amf = new(gnbctx.GnbAmf)
		amf.Log.Errorln("ran is nil")
		return
	}

	amf.Log.Traceln("Processing Ue Context Release Command")

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
		amf.Log.Errorln("Initiating Message is nil")
		return
	}

	ueCtxRelCmd := initiatingMessage.Value.UEContextReleaseCommand
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
		amf.Log.Errorln("No RANUENGAPID received")
		return
	}

	ngapId := ueNgapIds.UENGAPIDPair.RANUENGAPID.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		amf.Log.Errorln("No GnbUe found corresponding to RANUENGAPID:", ngapId)
		return
	}

	SendToGnbUe(gnbue, common.UE_CTX_RELEASE_COMMAND_EVENT, pdu, id)
}
