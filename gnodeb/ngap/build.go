// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package ngap

import (
	"encoding/hex"
	"fmt"
	"gnbsim/gnodeb/context"
	"gnbsim/util/ngapTestpacket"

	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapConvert"
	"github.com/free5gc/ngap/ngapType"
)

func GetNGSetupRequest(gnb *context.GNodeB) ([]byte, error) {

	message := ngapTestpacket.BuildNGSetupRequest()

	// GlobalRANNodeID
	ie := message.InitiatingMessage.Value.NGSetupRequest.ProtocolIEs.List[0]
	*(ie.Value.GlobalRANNodeID) = ngapConvert.RanIDToNgap(gnb.RanId)

	// RANNodeName
	ie = message.InitiatingMessage.Value.NGSetupRequest.ProtocolIEs.List[1]
	ie.Value.RANNodeName.Value = gnb.GnbName

	// TAC
	ie = message.InitiatingMessage.Value.NGSetupRequest.ProtocolIEs.List[2]

	supportedTaList := ie.Value.SupportedTAList
	// Clearing default entries.
	supportedTaList.List = nil

	for _, ta := range gnb.SupportedTaList {
		tac, err := hex.DecodeString(ta.Tac)
		if err != nil {
			gnb.Log.Errorln("DecodeString returned:", err)
			return nil, fmt.Errorf("invalid TAC")
		}
		supportedTaItem := ngapType.SupportedTAItem{}
		supportedTaItem.TAC.Value = tac

		broadcastPLMNList := &supportedTaItem.BroadcastPLMNList
		for _, plmnItem := range ta.BroadcastPLMNList {
			// BroadcastPLMNItem in BroadcastPLMNList
			broadcastPLMNItem := ngapType.BroadcastPLMNItem{}
			broadcastPLMNItem.PLMNIdentity = ngapConvert.PlmnIdToNgap(plmnItem.PlmnId)

			sliceSupportList := &broadcastPLMNItem.TAISliceSupportList
			for _, snssai := range plmnItem.TaiSliceSupportList {
				// SliceSupportItem in SliceSupportList
				sliceSupportItem := ngapType.SliceSupportItem{}
				sliceSupportItem.SNSSAI = ngapConvert.SNssaiToNgap(snssai)
				sliceSupportList.List = append(sliceSupportList.List, sliceSupportItem)
			}
			broadcastPLMNList.List = append(broadcastPLMNList.List, broadcastPLMNItem)
		}
		supportedTaList.List = append(supportedTaList.List, supportedTaItem)
	}

	return ngap.Encoder(message)
}

func GetUEContextReleaseRequest(gnbue *context.GnbCpUe) ([]byte, error) {
	var pduSessIds []int64
	f := func(k interface{}, v interface{}) bool {
		pduSessIds = append(pduSessIds, k.(int64))
		return true
	}

	gnbue.GnbUpUes.Range(f)

	message := ngapTestpacket.BuildUEContextReleaseRequest(gnbue.AmfUeNgapId,
		gnbue.GnbUeNgapId, pduSessIds)

	lst := message.InitiatingMessage.Value.UEContextReleaseRequest.ProtocolIEs.List

	// Cause
	ie := lst[len(lst)-1]
	ie.Value.Cause.RadioNetwork.Value = ngapType.CauseRadioNetworkPresentUserInactivity

	return ngap.Encoder(message)
}
