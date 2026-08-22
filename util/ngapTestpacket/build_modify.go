// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package ngapTestpacket

import (
	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/ngapType"
)

// QosFlowOutcome is what the gNB decided about one QoS flow it was asked to add or modify.
//
// A modification is not all-or-nothing at the radio: the gNB can admit some flows and refuse
// others, and the answer has to say which is which per flow. Carrying the outcome per flow rather
// than a single verdict is what lets a test drive a partial rejection.
type QosFlowOutcome struct {
	Cause     ngapType.Cause
	QfiValue  int64
	Succeeded bool
}

// BuildPDUSessionResourceModifyResponseTransfer builds the per-session transfer that reports the
// fate of each QoS flow.
//
// The DL and UL NG-U UP TNL Information fields are deliberately left absent. TS 38.413 makes them
// OPTIONAL, and a modification that changes QoS without moving the tunnel has nothing to say about
// them — which is the ordinary case and the one this simulator exists to produce.
func BuildPDUSessionResourceModifyResponseTransfer(outcomes []QosFlowOutcome) ([]byte, error) {
	transfer := ngapType.PDUSessionResourceModifyResponseTransfer{}

	var admitted []ngapType.QosFlowAddOrModifyResponseItem
	var refused []ngapType.QosFlowWithCauseItem
	for _, o := range outcomes {
		if o.Succeeded {
			item := ngapType.QosFlowAddOrModifyResponseItem{}
			item.QosFlowIdentifier.Value = o.QfiValue
			admitted = append(admitted, item)
			continue
		}
		item := ngapType.QosFlowWithCauseItem{}
		item.QosFlowIdentifier.Value = o.QfiValue
		item.Cause = o.Cause
		refused = append(refused, item)
	}

	if len(admitted) > 0 {
		transfer.QosFlowAddOrModifyResponseList = &ngapType.QosFlowAddOrModifyResponseList{List: admitted}
	}
	if len(refused) > 0 {
		transfer.QosFlowFailedToAddOrModifyList = &ngapType.QosFlowListWithCause{List: refused}
	}

	return aper.MarshalWithParams(transfer, "valueExt")
}

// BuildPDUSessionResourceModifyResponse builds the NGAP answer to a
// PDU SESSION RESOURCE MODIFY REQUEST.
//
// A session appears in the modify list when the gNB acted on it and in the failed list when it
// could not. Both lists are optional, and a response carrying neither is what an empty request
// produces.
func BuildPDUSessionResourceModifyResponse(amfUeNgapID, ranUeNgapID int64,
	modified map[int64][]byte, failed map[int64]ngapType.Cause,
) (pdu ngapType.NGAPPDU) {
	pdu.Present = ngapType.NGAPPDUPresentSuccessfulOutcome
	pdu.SuccessfulOutcome = new(ngapType.SuccessfulOutcome)

	successfulOutcome := pdu.SuccessfulOutcome
	successfulOutcome.ProcedureCode.Value = ngapType.ProcedureCodePDUSessionResourceModify
	successfulOutcome.Criticality.Value = ngapType.CriticalityPresentReject
	successfulOutcome.Value.Present = ngapType.SuccessfulOutcomePresentPDUSessionResourceModify
	successfulOutcome.Value.PDUSessionResourceModify = new(ngapType.PDUSessionResourceModifyResponse)

	response := successfulOutcome.Value.PDUSessionResourceModify
	ies := &response.ProtocolIEs

	ie := ngapType.PDUSessionResourceModifyResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDAMFUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceModifyResponseIEsPresentAMFUENGAPID
	ie.Value.AMFUENGAPID = new(ngapType.AMFUENGAPID)
	ie.Value.AMFUENGAPID.Value = amfUeNgapID
	ies.List = append(ies.List, ie)

	ie = ngapType.PDUSessionResourceModifyResponseIEs{}
	ie.Id.Value = ngapType.ProtocolIEIDRANUENGAPID
	ie.Criticality.Value = ngapType.CriticalityPresentIgnore
	ie.Value.Present = ngapType.PDUSessionResourceModifyResponseIEsPresentRANUENGAPID
	ie.Value.RANUENGAPID = new(ngapType.RANUENGAPID)
	ie.Value.RANUENGAPID.Value = ranUeNgapID
	ies.List = append(ies.List, ie)

	if len(modified) > 0 {
		ie = ngapType.PDUSessionResourceModifyResponseIEs{}
		ie.Id.Value = ngapType.ProtocolIEIDPDUSessionResourceModifyListModRes
		ie.Criticality.Value = ngapType.CriticalityPresentIgnore
		ie.Value.Present = ngapType.PDUSessionResourceModifyResponseIEsPresentPDUSessionResourceModifyListModRes
		ie.Value.PDUSessionResourceModifyListModRes = new(ngapType.PDUSessionResourceModifyListModRes)
		list := ie.Value.PDUSessionResourceModifyListModRes
		for pduSessID, transfer := range modified {
			item := ngapType.PDUSessionResourceModifyItemModRes{}
			item.PDUSessionID.Value = pduSessID
			item.PDUSessionResourceModifyResponseTransfer = transfer
			list.List = append(list.List, item)
		}
		ies.List = append(ies.List, ie)
	}

	if len(failed) > 0 {
		ie = ngapType.PDUSessionResourceModifyResponseIEs{}
		ie.Id.Value = ngapType.ProtocolIEIDPDUSessionResourceFailedToModifyListModRes
		ie.Criticality.Value = ngapType.CriticalityPresentIgnore
		ie.Value.Present = ngapType.PDUSessionResourceModifyResponseIEsPresentPDUSessionResourceFailedToModifyListModRes
		ie.Value.PDUSessionResourceFailedToModifyListModRes = new(ngapType.PDUSessionResourceFailedToModifyListModRes)
		list := ie.Value.PDUSessionResourceFailedToModifyListModRes
		for pduSessID, cause := range failed {
			unsuccessful := ngapType.PDUSessionResourceModifyUnsuccessfulTransfer{Cause: cause}
			encoded, err := aper.MarshalWithParams(unsuccessful, "valueExt")
			if err != nil {
				continue
			}
			item := ngapType.PDUSessionResourceFailedToModifyItemModRes{}
			item.PDUSessionID.Value = pduSessID
			item.PDUSessionResourceModifyUnsuccessfulTransfer = encoded
			list.List = append(list.List, item)
		}
		ies.List = append(ies.List, ie)
	}

	return pdu
}
