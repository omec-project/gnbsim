// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"fmt"

	"github.com/free5gc/amf/context"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
)

// HandleNGSetupResponse processes the NG Setup Response and updates GnbAmf
// context
func HandleNGSetupResponse(amf *GnbAmf, pdu *ngapType.NGAPPDU) {
	fmt.Printf("decoded NGSETUP RESPONSE: %#v", pdu)
	var amfName *ngapType.AMFName
	var servedGUAMIList *ngapType.ServedGUAMIList
	var relativeAMFCapacity *ngapType.RelativeAMFCapacity
	var plmnSupportList *ngapType.PLMNSupportList
	// TODO Process optional IEs

	if amf == nil {
		fmt.Println("ran is nil")
		return
	}
	if pdu == nil {
		fmt.Println("NGAP Message is nil")
		return
	}
	successfulOutcome := pdu.SuccessfulOutcome
	if successfulOutcome == nil {
		fmt.Println("Initiating Message is nil")
		return
	}
	nGSetupResponse := successfulOutcome.Value.NGSetupResponse
	if nGSetupResponse == nil {
		fmt.Println("NGSetupResponse is nil")
		return
	}

	fmt.Println("Handle NG Setup response")
	for i := 0; i < len(nGSetupResponse.ProtocolIEs.List); i++ {
		ie := nGSetupResponse.ProtocolIEs.List[i]
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFName:
			amfName = ie.Value.AMFName
			fmt.Println("Decode IE AMFName")
			if amfName == nil {
				fmt.Println("AMFName is nil")
				return
			}
		case ngapType.ProtocolIEIDServedGUAMIList:
			servedGUAMIList = ie.Value.ServedGUAMIList
			fmt.Println("Decode IE ServedGUAMIList")
			if servedGUAMIList == nil {
				fmt.Println("ServedGUAMIList is nil")
				return
			}
		case ngapType.ProtocolIEIDRelativeAMFCapacity:
			relativeAMFCapacity = ie.Value.RelativeAMFCapacity
			fmt.Println("Decode IE RelativeAMFCapacity")
			if relativeAMFCapacity == nil {
				fmt.Println("RelativeAMFCapacity is nil")
				return
			}
		case ngapType.ProtocolIEIDPLMNSupportList:
			plmnSupportList = ie.Value.PLMNSupportList
			fmt.Println("Decode IE PLMNSupportList")
			if plmnSupportList == nil {
				fmt.Println("PLMNSupportList is nil")
				return
			}
		}
	}

	amf.SetAMFName(amfName.Value)
	amf.SetRelativeAMFCapacity(relativeAMFCapacity.Value)

	// Clearing any existing contents of ServedGuamiList within GnbAmf
	if len(amf.ServedGuamiList) != 0 {
		amf.ServedGuamiList = NewServedGUAMIList()
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
		fmt.Println("NG Setup Response : Empty ServedGuamiList received")
	} else {
		// TODO: Need to check
	}

	// Clearing any existing contents of PlmnSupportList within GnbAmf
	if len(amf.PlmnSupportList) != 0 {
		amf.PlmnSupportList = NewPlmnSupportList()
	}
	capOfPlmnSupportList := cap(amf.PlmnSupportList)
	for _, plmnSupportItem := range plmnSupportList.List {
		plmnSI := context.NewPlmnSupportItem()

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
		fmt.Println("NG Setup Response : Empty PLMNSupportList received")
	} else {
		// TODO: Need to check whether it should be compared against some
		// existing list within gNodeB
	}

	fmt.Println("Processed NG Setup Response")
}
