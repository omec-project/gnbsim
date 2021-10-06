// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbamfworker

import (
	"fmt"
	"log"

	"gnbsim/common"

	"gnbsim/gnodeb/context"

	amfctx "github.com/free5gc/amf/context"
	"github.com/free5gc/aper"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
)

// HandleNGSetupResponse processes the NG Setup Response and updates GnbAmf
// context
func HandleNgSetupResponse(amf *context.GnbAmf, pdu *ngapType.NGAPPDU) {
	fmt.Printf("decoded NGSETUP RESPONSE: %#v\n", pdu)
	var amfName *ngapType.AMFName
	var servedGUAMIList *ngapType.ServedGUAMIList
	var relativeAMFCapacity *ngapType.RelativeAMFCapacity
	var plmnSupportList *ngapType.PLMNSupportList
	// TODO Process optional IEs

	if amf == nil {
		log.Println("ran is nil")
		return
	}
	if pdu == nil {
		log.Println("NGAP Message is nil")
		return
	}
	successfulOutcome := pdu.SuccessfulOutcome
	if successfulOutcome == nil {
		log.Println("Initiating Message is nil")
		return
	}
	ngSetupResponse := successfulOutcome.Value.NGSetupResponse
	if ngSetupResponse == nil {
		log.Println("NGSetupResponse is nil")
		return
	}

	log.Println("Handle NG Setup response")
	for i := 0; i < len(ngSetupResponse.ProtocolIEs.List); i++ {
		ie := ngSetupResponse.ProtocolIEs.List[i]
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFName:
			amfName = ie.Value.AMFName
			log.Println("Decode IE AMFName")
			if amfName == nil {
				log.Println("AMFName is nil")
				return
			}
		case ngapType.ProtocolIEIDServedGUAMIList:
			servedGUAMIList = ie.Value.ServedGUAMIList
			log.Println("Decode IE ServedGUAMIList")
			if servedGUAMIList == nil {
				log.Println("ServedGUAMIList is nil")
				return
			}
		case ngapType.ProtocolIEIDRelativeAMFCapacity:
			relativeAMFCapacity = ie.Value.RelativeAMFCapacity
			log.Println("Decode IE RelativeAMFCapacity")
			if relativeAMFCapacity == nil {
				log.Println("RelativeAMFCapacity is nil")
				return
			}
		case ngapType.ProtocolIEIDPLMNSupportList:
			plmnSupportList = ie.Value.PLMNSupportList
			log.Println("Decode IE PLMNSupportList")
			if plmnSupportList == nil {
				log.Println("PLMNSupportList is nil")
				return
			}
		}
	}

	amf.SetAMFName(amfName.Value)
	amf.SetRelativeAMFCapacity(relativeAMFCapacity.Value)

	// Initializing the ServedGuamiList slice in GnbAmf if not already initialized
	// This will also clear any existing contents of ServedGuamiList within GnbAmf
	if len(amf.ServedGuamiList) != 0 || cap(amf.ServedGuamiList) == 0 {
		amf.ServedGuamiList = context.NewServedGUAMIList()
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
		log.Println("NG Setup Response : Empty ServedGuamiList received")
	} /* else {
		// TODO: Need to check
	}*/

	// Initializing the PlmnSuportList slice in GnbAmf if not already initialized
	// This will also clear any existing contents of PlmnSupportList within GnbAmf
	if len(amf.PlmnSupportList) != 0 || cap(amf.PlmnSupportList) == 0 {
		amf.PlmnSupportList = context.NewPlmnSupportList()
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
		log.Println("NG Setup Response : Empty PLMNSupportList received")
	} /*else {
		// TODO: Need to check whether it should be compared against some
		// existing list within gNodeB
	}*/

	amf.SetNgSetupStatus(true)
	log.Println("Processed NG Setup Response")
}

func HandleNgSetupFailure(amf *context.GnbAmf, pdu *ngapType.NGAPPDU) {
	var cause *ngapType.Cause

	if amf == nil {
		log.Println("ran is nil")
		return
	}
	if pdu == nil {
		log.Println("NGAP Message is nil")
		return
	}
	UnSuccessfulOutcome := pdu.UnsuccessfulOutcome
	if UnSuccessfulOutcome == nil {
		log.Println("UnSuccessfulOutcome Message is nil")
		return
	}
	ngSetupFailure := UnSuccessfulOutcome.Value.NGSetupFailure
	if ngSetupFailure == nil {
		log.Println("NGSetupResponse is nil")
		return
	}

	log.Println("Handle NG Setup Failure")
	for i := 0; i < len(ngSetupFailure.ProtocolIEs.List); i++ {
		ie := ngSetupFailure.ProtocolIEs.List[i]
		if ie.Id.Value == ngapType.ProtocolIEIDCause {
			cause = ie.Value.Cause
			log.Println("Decode IE Cause")
			if cause == nil {
				log.Println("Cause is nil")
				return
			}
		}
		// TODO handle TimeToWait IE
	}

	PrintAndGetCause(cause)
	amf.SetNgSetupStatus(false)

	log.Println("Processed NG Setup Failure")
}

func HandleDownlinkNasTransport(gnb *context.GNodeB, amf *context.GnbAmf, pdu *ngapType.NGAPPDU) {
	var gnbUeNgapId *ngapType.RANUENGAPID

	if amf == nil {
		log.Println("ran is nil")
		return
	}
	if pdu == nil {
		log.Println("NGAP Message is nil")
		return
	}
	if gnb == nil {
		log.Println("GNodeB Message is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		log.Println("Initiating Message is nil")
		return
	}
	downlinkNasTransport := initiatingMessage.Value.DownlinkNASTransport
	if downlinkNasTransport == nil {
		log.Println("DownlinkNASTransport is nil")
		return
	}

	log.Println("Handle Downlink NAS Transport")
	for i := 0; i < len(downlinkNasTransport.ProtocolIEs.List); i++ {
		ie := downlinkNasTransport.ProtocolIEs.List[i]
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			log.Println("Decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				log.Println("RANUENGAPID is nil")
				return
			}
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		log.Println("No GnbUe found corresponding to RANUENGAPID:", ngapId)
		return
	}

	SendToGnbUe(gnbue, common.DOWNLINK_NAS_TRANSPORT_EVENT, pdu)
}

func HandleInitialContextSetupRequest(gnb *context.GNodeB, amf *context.GnbAmf, pdu *ngapType.NGAPPDU) {
	var gnbUeNgapId *ngapType.RANUENGAPID

	if amf == nil {
		log.Println("ran is nil")
		return
	}
	if pdu == nil {
		log.Println("NGAP Message is nil")
		return
	}
	if gnb == nil {
		log.Println("GNodeB Message is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		log.Println("Initiating Message is nil")
		return
	}
	initialContextSetupRequest := initiatingMessage.Value.InitialContextSetupRequest
	if initialContextSetupRequest == nil {
		log.Println("InitialContextSetupRequest is nil")
		return
	}

	log.Println("InitialContextSetupRequest")
	for _, ie := range initialContextSetupRequest.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			log.Println("Decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				log.Println("RANUENGAPID is nil")
				return
			}
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		log.Println("No GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.INITIAL_CONTEXT_SETUP_REQUEST_EVENT, pdu)
}

// TODO : Much of the code is repeated in each handler
func HandlePduSessResourceSetupRequest(gnb *context.GNodeB, amf *context.GnbAmf, pdu *ngapType.NGAPPDU) {
	var gnbUeNgapId *ngapType.RANUENGAPID

	if amf == nil {
		log.Println("ran is nil")
		return
	}
	if pdu == nil {
		log.Println("NGAP Message is nil")
		return
	}
	if gnb == nil {
		log.Println("GNodeB Message is nil")
		return
	}
	initiatingMessage := pdu.InitiatingMessage
	if initiatingMessage == nil {
		log.Println("Initiating Message is nil")
		return
	}
	pduSessResourceSetupReq := initiatingMessage.Value.PDUSessionResourceSetupRequest
	if pduSessResourceSetupReq == nil {
		log.Println("InitialContextSetupRequest is nil")
		return
	}

	log.Println("InitialContextSetupRequest")
	for _, ie := range pduSessResourceSetupReq.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDRANUENGAPID {
			gnbUeNgapId = ie.Value.RANUENGAPID
			log.Println("Decode IE RANUENGAPID")
			if gnbUeNgapId == nil {
				log.Println("RANUENGAPID is nil")
				return
			}
		}
	}
	ngapId := gnbUeNgapId.Value
	gnbue := gnb.GnbUes.GetGnbCpUe(ngapId)
	if gnbue == nil {
		log.Println("No GnbUe found corresponding to RANUENGAPID:")
		return
	}

	SendToGnbUe(gnbue, common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT, pdu)
}

func PrintAndGetCause(cause *ngapType.Cause) (present int, value aper.Enumerated) {
	present = cause.Present
	switch cause.Present {
	case ngapType.CausePresentRadioNetwork:
		fmt.Printf("Cause RadioNetwork[%d]\n", cause.RadioNetwork.Value)
		value = cause.RadioNetwork.Value
	case ngapType.CausePresentTransport:
		fmt.Printf("Cause Transport[%d]\n", cause.Transport.Value)
		value = cause.Transport.Value
	case ngapType.CausePresentProtocol:
		fmt.Printf("Cause Protocol[%d]\n", cause.Protocol.Value)
		value = cause.Protocol.Value
	case ngapType.CausePresentNas:
		fmt.Printf("Cause Nas[%d]\n", cause.Nas.Value)
		value = cause.Nas.Value
	case ngapType.CausePresentMisc:
		fmt.Printf("Cause Misc[%d]\n", cause.Misc.Value)
		value = cause.Misc.Value
	default:
		fmt.Printf("Invalid Cause group[%d]\n", cause.Present)
	}
	return
}
