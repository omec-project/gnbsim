// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
	"github.com/omec-project/ngap/v2"
	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/ngapType"
)

// HandlePduSessResourceModifyRequest answers the network's request to modify a PDU session.
//
// Three things happen, and the order matters. The gNB decides which QoS flows it will admit and
// records them, so its own view of the session matches what it is about to report. The NAS
// container, which carries the PDU SESSION MODIFICATION COMMAND, is passed to the UE. Then the
// response goes back to the AMF.
//
// The NAS message is forwarded before the response is sent because the UE's answer and the gNB's
// answer are independent: the core is entitled to receive them in either order, and delaying the
// NAS behind the NGAP response would make this simulator quietly stricter than a real gNB.
func HandlePduSessResourceModifyRequest(gnbue *gnbctx.GnbCpUe, intfcMsg common.InterfaceMessage) {
	msg := intfcMsg.(*common.N2Message)
	pdu := msg.NgapPdu
	if pdu == nil || pdu.InitiatingMessage == nil {
		gnbue.Log.Errorln("PDU Session Resource Modify Request is nil")
		return
	}
	modifyRequest := pdu.InitiatingMessage.Value.PDUSessionResourceModify
	if modifyRequest == nil {
		gnbue.Log.Errorln("PDUSessionResourceModifyRequest is nil")
		return
	}

	var modifyList *ngapType.PDUSessionResourceModifyListModReq
	for _, ie := range modifyRequest.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDPDUSessionResourceModifyListModReq {
			modifyList = ie.Value.PDUSessionResourceModifyListModReq
		}
	}
	// An empty request is still answered. The response carries neither list, which is exactly
	// what BuildPDUSessionResourceModifyResponse produces for one -- whereas returning here left
	// the core waiting for a message that never came, and a timeout on its side reads as the
	// core's defect rather than as a request that named nothing.
	var items []ngapType.PDUSessionResourceModifyItemModReq
	if modifyList != nil {
		items = modifyList.List
	}

	if len(items) == 0 {
		gnbue.Log.Warnln("PDU Session Resource Modify Request names no sessions; answering with an empty response")
	}

	modified := make(map[int64][]byte)
	failed := make(map[int64]ngapType.Cause)
	// Keyed by session, and only sent for sessions the gNB actually acted on. See below.
	pendingNas := make(map[int64][]byte)

	for _, item := range items {
		pduSessID := item.PDUSessionID.Value

		if item.NASPDU != nil && item.NASPDU.Value != nil {
			pendingNas[pduSessID] = item.NASPDU.Value
		}

		if gnbue.Gnb.ModifyRejectAll {
			// The whole session is refused. TS 38.413 puts this in the failed list with a cause
			// rather than in the modify list, and the core treats it as a delivery failure for
			// the modification as a whole.
			gnbue.Log.Infoln("refusing the whole modification for PDU session", pduSessID,
				"as configured")
			failed[pduSessID] = radioResourcesNotAvailable()
			continue
		}

		transfer := ngapType.PDUSessionResourceModifyRequestTransfer{}
		if err := aper.UnmarshalWithParams(item.PDUSessionResourceModifyRequestTransfer,
			&transfer, "valueExt"); err != nil {
			gnbue.Log.Errorln("failed to decode the modify request transfer:", err)
			failed[pduSessID] = radioResourcesNotAvailable()
			continue
		}

		outcomes := decideQosFlows(gnbue, pduSessID, &transfer)
		if len(outcomes) == 0 {
			// Nothing to decide — a modification that changes only session-level parameters. The
			// session is still reported as modified, with an answer that names no flows.
			gnbue.Log.Infoln("modification for PDU session", pduSessID, "names no QoS flows")
		}

		encoded, err := ngapTestpacket.BuildPDUSessionResourceModifyResponseTransfer(outcomes)
		if err != nil {
			gnbue.Log.Errorln("failed to encode the modify response transfer:", err)
			failed[pduSessID] = radioResourcesNotAvailable()
			continue
		}
		modified[pduSessID] = encoded
	}

	// The NAS container goes to the UE only for a session the gNB acted on.
	//
	// A session it refused outright has had nothing changed at the radio, so telling the UE its
	// QoS changed would leave the UE applying parameters that do not exist — the same divergence
	// the core's realignment procedure exists to repair, manufactured by the gNB itself. A
	// partially accepted session still gets the container: the UE needs to know about the flows
	// that were admitted, and the core corrects the rest.
	//
	// Observed before this was fixed: the gNB refused the only flow, forwarded the command
	// anyway, and the UE acknowledged a modification that the core had already abandoned.
	var nasPdus common.NasPduList
	for pduSessID, nas := range pendingNas {
		if _, acted := modified[pduSessID]; !acted {
			gnbue.Log.Infoln("withholding the modification command for PDU session", pduSessID,
				"because nothing was admitted for it")
			continue
		}
		nasPdus = append(nasPdus, nas)
	}
	if len(nasPdus) > 0 {
		SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nasPdus, msg.Id)
		gnbue.Log.Debugln("sent the modification command to the UE")
	}

	responsePdu, err := ngapTestpacket.BuildPDUSessionResourceModifyResponse(gnbue.AmfUeNgapId,
		gnbue.GnbUeNgapId, modified, failed)
	if err != nil {
		gnbue.Log.Errorln("failed to build the PDU Session Resource Modify Response:", err)
		return
	}
	encoded, err := ngap.Encoder(responsePdu)
	if err != nil {
		gnbue.Log.Errorln("failed to encode the PDU Session Resource Modify Response:", err)
		return
	}
	if err := gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, encoded, 0); err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}
	gnbue.Log.Infoln("sent PDU Session Resource Modify Response:",
		len(modified), "session(s) modified,", len(failed), "failed")
}

// radioResourcesNotAvailable is the cause a gNB gives when it will not admit what was asked of it.
func radioResourcesNotAvailable() ngapType.Cause {
	cause := ngapType.Cause{}
	cause.Present = ngapType.CausePresentRadioNetwork
	cause.RadioNetwork = &ngapType.CauseRadioNetwork{
		Value: ngapType.CauseRadioNetworkPresentRadioResourcesNotAvailable,
	}
	return cause
}

// decideQosFlows records what the gNB will do with each QoS flow the request names, and returns
// the per-flow outcome to report.
//
// A refused flow is still recorded as refused rather than skipped: the answer has to name it, or
// the core cannot tell the difference between a flow the radio would not admit and one the request
// never mentioned.
func decideQosFlows(gnbue *gnbctx.GnbCpUe, pduSessID int64,
	transfer *ngapType.PDUSessionResourceModifyRequestTransfer,
) []ngapTestpacket.QosFlowOutcome {
	var requested *ngapType.QosFlowAddOrModifyRequestList
	for _, ie := range transfer.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDQosFlowAddOrModifyRequestList {
			requested = ie.Value.QosFlowAddOrModifyRequestList
		}
	}
	if requested == nil {
		return nil
	}

	refuse := make(map[int64]bool, len(gnbue.Gnb.ModifyRejectQfis))
	for _, qfi := range gnbue.Gnb.ModifyRejectQfis {
		refuse[qfi] = true
	}

	upCtx, err := gnbue.GetGnbUpUe(pduSessID)
	if err != nil {
		gnbue.Log.Warnln("no user plane context for PDU session", pduSessID,
			"so the admitted flows cannot be recorded:", err)
	}

	outcomes := make([]ngapTestpacket.QosFlowOutcome, 0, len(requested.List))
	for _, item := range requested.List {
		qfi := item.QosFlowIdentifier.Value
		if refuse[qfi] {
			gnbue.Log.Infoln("refusing QoS flow", qfi, "on PDU session", pduSessID, "as configured")
			outcomes = append(outcomes, ngapTestpacket.QosFlowOutcome{
				QfiValue: qfi, Cause: radioResourcesNotAvailable(),
			})
			continue
		}

		// Admitted, so the gNB's own view of the session has to carry it. Recording only on
		// establishment is what would let the gNB report a flow it is not actually serving.
		if upCtx != nil && upCtx.QosFlows != nil {
			upCtx.QosFlows[qfi] = &ngapType.QosFlowSetupRequestItem{
				QosFlowIdentifier:         item.QosFlowIdentifier,
				QosFlowLevelQosParameters: qosParamsOrZero(item.QosFlowLevelQosParameters),
			}
		}
		gnbue.Log.Infoln("admitted QoS flow", qfi, "on PDU session", pduSessID)
		outcomes = append(outcomes, ngapTestpacket.QosFlowOutcome{QfiValue: qfi, Succeeded: true})
	}
	return outcomes
}

// qosParamsOrZero keeps the recorded flow usable when the request modified a flow without
// restating its parameters, which is permitted.
func qosParamsOrZero(p *ngapType.QosFlowLevelQosParameters) ngapType.QosFlowLevelQosParameters {
	if p == nil {
		return ngapType.QosFlowLevelQosParameters{}
	}
	return *p
}
