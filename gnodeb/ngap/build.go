// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package ngap

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/omec-project/aper"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
	"github.com/omec-project/ngap"
	"github.com/omec-project/ngap/ngapConvert"
	"github.com/omec-project/ngap/ngapType"
)

func GetNGSetupRequest(gnb *gnbctx.GNodeB) ([]byte, error) {
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

// GetInitialUEMessage encodes NGAP InitialUEMessage for the given UE.
//
//	gnbue: gNB-UE context.
//	nasPdu: value of id-NAS-PDU from the UE.
func GetInitialUEMessage(gnbue *gnbctx.GnbCpUe, nasPdu []byte, tmsi string) ([]byte, error) {
	message := ngapTestpacket.BuildInitialUEMessage(gnbue.GnbUeNgapId, nasPdu, tmsi)
	ies := message.InitiatingMessage.Value.InitialUEMessage.ProtocolIEs.List

	if e := updateUserLocationInformation(gnbue.Gnb, ies[2].Value.UserLocationInformation); e != nil {
		return nil, e
	}

	return ngap.Encoder(message)
}

// GetUplinkNASTransport encodes NGAP UplinkNASTransport for the given UE.
//
//	gnbue: gNB-UE context.
//	nasPdu: value of id-NAS-PDU from the UE.
func GetUplinkNASTransport(gnbue *gnbctx.GnbCpUe, nasPdu []byte) ([]byte, error) {
	message := ngapTestpacket.BuildUplinkNasTransport(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, nasPdu)
	ies := message.InitiatingMessage.Value.UplinkNASTransport.ProtocolIEs.List

	if e := updateUserLocationInformation(gnbue.Gnb, ies[3].Value.UserLocationInformation); e != nil {
		return nil, e
	}

	return ngap.Encoder(message)
}

func GetUEContextReleaseRequest(gnbue *gnbctx.GnbCpUe) ([]byte, error) {
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

// GetUplinkNASTransport encodes NGAP UEContextReleaseComplete for the given UE.
//
//	gnbue: gNB-UE context.
//	nasPdu: value of id-NAS-PDU from the UE.
func GetUEContextReleaseComplete(gnbue *gnbctx.GnbCpUe) ([]byte, error) {
	var pduSessIds []int64
	gnbue.GnbUpUes.Range(func(k interface{}, v interface{}) bool {
		pduSessIds = append(pduSessIds, k.(int64))
		return true
	})

	message := ngapTestpacket.BuildUEContextReleaseComplete(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, pduSessIds)
	ies := message.SuccessfulOutcome.Value.UEContextReleaseComplete.ProtocolIEs.List

	if e := updateUserLocationInformation(gnbue.Gnb, ies[2].Value.UserLocationInformation); e != nil {
		return nil, e
	}

	return ngap.Encoder(message)
}

// updateUserLocationInformation updates UserLocationInformation element to match gNB information.
//
//	gnb: gNB context.
//	uli: UserLocationInformation prepared by ngapTestpacket package.
func updateUserLocationInformation(gnb *gnbctx.GNodeB, uli *ngapType.UserLocationInformation) error {
	nr := uli.UserLocationInformationNR

	nr.NRCGI.PLMNIdentity = ngapConvert.PlmnIdToNgap(*gnb.RanId.PlmnId)
	nr.TAI.PLMNIdentity = nr.NRCGI.PLMNIdentity

	gnbID, e := strconv.ParseUint(gnb.RanId.GNbId.GNBValue, 16, 64)
	if e != nil {
		return fmt.Errorf("invalid GNB ID: %w", e)
	}
	// NRCI contains gnbID and cellID, here we assume cellID is zero
	nrci := gnbID << uint64(36-gnb.RanId.GNbId.BitLength)
	nrciBuf := [8]byte{}
	binary.BigEndian.PutUint64(nrciBuf[:], nrci)
	nr.NRCGI.NRCellIdentity.Value.Bytes = nrciBuf[3:]

	if len(gnb.SupportedTaList) < 1 {
		return errors.New("unexpected empty SupportedTaList")
	}
	tac, e := hex.DecodeString(gnb.SupportedTaList[0].Tac)
	if e != nil {
		return fmt.Errorf("invalid TAC: %w", e)
	}
	nr.TAI.TAC.Value = aper.OctetString(tac)

	return nil
}
