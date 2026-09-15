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
	// Keyed by session, and only sent for sessions the gNB reports as modified. See below.
	pendingNas := make(map[int64][]byte)

	for i := range items {
		item := &items[i]
		pduSessID := item.PDUSessionID.Value

		if item.NASPDU != nil && item.NASPDU.Value != nil {
			pendingNas[pduSessID] = item.NASPDU.Value
		}

		encoded, cause := modifySession(gnbue, item)
		if cause != nil {
			failed[pduSessID] = *cause
			continue
		}
		modified[pduSessID] = encoded
	}

	// The NAS container goes to the UE only for a session reported as modified. That is a decision
	// about the session, not about its flows.
	//
	// A session the gNB failed as a whole — refused by modifyRejectAll, one it holds no context
	// for, or one whose transfer would not decode or encode — has had nothing changed at the
	// radio, so telling the UE its QoS changed would leave the UE applying parameters that do not
	// exist: the same divergence the core's realignment procedure exists to repair, manufactured
	// by the gNB itself.
	//
	// TS 38.413 clause 8.2.3.2 ties the two together: the NAS-PDU is passed to the UE "only if at
	// least one of the requests included in the PDU Session Resource Modify Request Transfer IE is
	// successful (i.e. the PDU session is included in the PDU Session Resource Modify Response
	// Item IE ...)" — which is this membership test.
	//
	// Note what that condition is not: it is not "at least one QoS flow was admitted". The SMF
	// sends a session AMBR in the same transfer as the flow list, and that request succeeds
	// whatever the radio decides about the flows — so refusing every flow still leaves the
	// session successfully modified, and the command still goes to the UE. This gNB reaches the
	// same answer by a blunter route: any transfer it could decode leaves the session in the
	// modify list, since it evaluates the flow list and nothing else. A transfer carrying only an
	// add-or-modify list whose flows were all refused would therefore be reported as successful
	// where a stricter gNB would fail the session. No core in this stack sends that shape.
	var nasPdus common.NasPduList
	for pduSessID, nas := range pendingNas {
		if _, wasModified := modified[pduSessID]; !wasModified {
			gnbue.Log.Infoln("withholding the modification command for PDU session", pduSessID,
				"because the gNB failed the session as a whole")
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

// modifySession decides what happens to one session named by the request, and returns either the
// response transfer to report it modified with, or the cause to report it failed with.
//
// The gNB's own view of the session is changed last, once the answer has been encoded. A session
// that fails anywhere in here is reported as failed and its modification command withheld, so the
// core and the UE both go on holding it at its previous parameters — and a gNB that had already
// released or admitted flows would be the only party that had moved.
func modifySession(gnbue *gnbctx.GnbCpUe, item *ngapType.PDUSessionResourceModifyItemModReq,
) ([]byte, *ngapType.Cause) {
	pduSessID := item.PDUSessionID.Value

	// The session has to be one this gNB is serving, and that is settled first. Without a user
	// plane context there is no bearer to admit a flow onto, and answering "admitted" would
	// promise the core a radio resource that does not exist — the session would carry flows at
	// the SMF that the gNB cannot serve, and the UE would be told its QoS changed.
	//
	// It is decided ahead of the refusal and the decode because it is a fact about the request's
	// target rather than about its contents: the PDU Session ID is on the item, not inside the
	// transfer, so neither a configured refusal nor a transfer that will not decode changes the
	// answer. Behind them, a session this gNB never heard of was reported as
	// "radio resources not available", which tells the core to retry later for something that
	// will never exist -- and the configured refusal is there to exercise a modification both
	// sides hold being refused, not to describe a session that does not exist.
	upCtx, err := gnbue.GetGnbUpUe(pduSessID)
	if err != nil {
		gnbue.Log.Errorln("no user plane context for PDU session", pduSessID,
			"so the modification cannot be admitted:", err)
		return nil, causePtr(unknownPduSessionID())
	}

	if gnbue.Gnb.ModifyRejectAll {
		// The whole session is refused. TS 38.413 puts this in the failed list with a cause
		// rather than in the modify list, and the core treats it as a delivery failure for
		// the modification as a whole.
		gnbue.Log.Infoln("refusing the whole modification for PDU session", pduSessID,
			"as configured")
		return nil, causePtr(radioResourcesNotAvailable())
	}

	transfer := ngapType.PDUSessionResourceModifyRequestTransfer{}
	if err = aper.UnmarshalWithParams(item.PDUSessionResourceModifyRequestTransfer,
		&transfer, "valueExt"); err != nil {
		gnbue.Log.Errorln("failed to decode the modify request transfer:", err)
		return nil, causePtr(radioResourcesNotAvailable())
	}

	// A request that moves the session's uplink tunnel is refused rather than answered. The
	// UL NG-U UP TNL Modify List names the tunnel the gNB is to send uplink to from now on, and
	// TS 38.413 clause 8.2.3.2 has the NG-RAN node use it and report the new endpoint back in the
	// response transfer. This gNB does neither: it would keep sending to the tunnel it has while
	// telling the core the modification succeeded, so the uplink would arrive nowhere and the core
	// would have no reason to look. Failing the session says what is true -- the gNB did not make
	// the change -- and leaves the session on its old parameters at both ends.
	//
	// No core in this stack sends the IE, so this is the answer to a request that has not been
	// seen rather than a path in use.
	if modifiesTheUplinkTunnel(&transfer) {
		gnbue.Log.Errorln("refusing the modification for PDU session", pduSessID,
			"because it moves the uplink tunnel, which this gNB does not do")
		return nil, causePtr(unspecifiedRadioNetworkFailure())
	}

	plan := decideQosFlows(gnbue, pduSessID, &transfer)
	if len(plan.outcomes) == 0 {
		// Nothing to decide: the request names no QoS flows to add or modify, so the session
		// is reported as modified with an answer that names none. A request carrying only a
		// QoS Flow to Release List arrives here too — the response has no list to report its
		// releases in, and withdrawing a flow is a modification that succeeded.
		gnbue.Log.Infoln("modification for PDU session", pduSessID,
			"names no QoS flows to add or modify")
	}

	encoded, err := ngapTestpacket.BuildPDUSessionResourceModifyResponseTransfer(plan.outcomes)
	if err != nil {
		gnbue.Log.Errorln("failed to encode the modify response transfer:", err)
		return nil, causePtr(radioResourcesNotAvailable())
	}

	plan.apply(upCtx)
	return encoded, nil
}

// causePtr distinguishes "this session failed, with this cause" from "it did not".
func causePtr(cause ngapType.Cause) *ngapType.Cause {
	return &cause
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

// unknownPduSessionID is the cause for a request naming a session this gNB is not serving.
func unknownPduSessionID() ngapType.Cause {
	cause := ngapType.Cause{}
	cause.Present = ngapType.CausePresentRadioNetwork
	cause.RadioNetwork = &ngapType.CauseRadioNetwork{
		Value: ngapType.CauseRadioNetworkPresentUnknownPDUSessionID,
	}
	return cause
}

// modifiesTheUplinkTunnel reports whether the request asks the gNB to send uplink somewhere new.
func modifiesTheUplinkTunnel(transfer *ngapType.PDUSessionResourceModifyRequestTransfer) bool {
	for _, ie := range transfer.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDULNGUUPTNLModifyList &&
			ie.Value.ULNGUUPTNLModifyList != nil {
			return true
		}
	}
	return false
}

// unspecifiedRadioNetworkFailure is the cause for a request this gNB will not carry out, where no
// more specific cause describes it: the radio network layer refused, and the log says why.
func unspecifiedRadioNetworkFailure() ngapType.Cause {
	cause := ngapType.Cause{}
	cause.Present = ngapType.CausePresentRadioNetwork
	cause.RadioNetwork = &ngapType.CauseRadioNetwork{
		Value: ngapType.CauseRadioNetworkPresentUnspecified,
	}
	return cause
}

// qosFlowPlan is what the gNB has decided to do with a session's QoS flows: the per-flow outcomes
// to report, and the changes to its own view of the session. The two are kept apart so the second
// can wait until the first has encoded.
type qosFlowPlan struct {
	outcomes []ngapTestpacket.QosFlowOutcome
	admitted []admittedQosFlow
	released []int64
}

// admittedQosFlow is a flow to record on the session, under the identity it is recorded by.
type admittedQosFlow struct {
	item *ngapType.QosFlowSetupRequestItem
	qfi  int64
}

// apply writes the plan to the gNB's view of the session.
//
// Releases go first, because a request may carry nothing else: the SMF builds a release-only
// transfer when it withdraws a flow.
func (p *qosFlowPlan) apply(upCtx *gnbctx.GnbUpUe) {
	for _, qfi := range p.released {
		upCtx.RemoveQosFlow(qfi)
	}
	for _, flow := range p.admitted {
		upCtx.AddQosFlow(flow.qfi, flow.item)
	}
}

// decideQosFlows works out what the gNB will do with each QoS flow the request names. It decides
// and reports; it changes nothing, which is what lets the decision be discarded if the answer
// cannot be encoded.
//
// A refused flow is still recorded as refused rather than skipped: the answer has to name it, or
// the core cannot tell the difference between a flow the radio would not admit and one the request
// never mentioned. A released flow is not reported at all -- the response has no list for it --
// but it is dropped from the gNB's view of the session, which is what TS 38.413 clause 8.2.3.2
// asks for when it has the NG-RAN node de-associate a released flow from its bearer.
func decideQosFlows(gnbue *gnbctx.GnbCpUe, pduSessID int64,
	transfer *ngapType.PDUSessionResourceModifyRequestTransfer,
) qosFlowPlan {
	plan := qosFlowPlan{released: releasedQfis(transfer)}
	for _, qfi := range plan.released {
		gnbue.Log.Infoln("releasing QoS flow", qfi, "on PDU session", pduSessID)
	}

	var requested *ngapType.QosFlowAddOrModifyRequestList
	for _, ie := range transfer.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDQosFlowAddOrModifyRequestList {
			requested = ie.Value.QosFlowAddOrModifyRequestList
		}
	}
	if requested == nil {
		return plan
	}

	refuse := make(map[int64]bool, len(gnbue.Gnb.ModifyRejectQfis))
	for _, qfi := range gnbue.Gnb.ModifyRejectQfis {
		refuse[qfi] = true
	}

	plan.outcomes = make([]ngapTestpacket.QosFlowOutcome, 0, len(requested.List))
	for _, item := range requested.List {
		qfi := item.QosFlowIdentifier.Value
		if refuse[qfi] {
			gnbue.Log.Infoln("refusing QoS flow", qfi, "on PDU session", pduSessID, "as configured")
			plan.outcomes = append(plan.outcomes, ngapTestpacket.QosFlowOutcome{
				QfiValue: qfi, Cause: radioResourcesNotAvailable(),
			})
			continue
		}

		// Admitted, so the gNB's own view of the session has to carry it. Recording only on
		// establishment is what would let the gNB report a flow it is not actually serving.
		plan.admitted = append(plan.admitted, admittedQosFlow{
			qfi: qfi,
			item: &ngapType.QosFlowSetupRequestItem{
				QosFlowIdentifier:         item.QosFlowIdentifier,
				QosFlowLevelQosParameters: qosParamsOrZero(item.QosFlowLevelQosParameters),
			},
		})
		gnbue.Log.Infoln("admitted QoS flow", qfi, "on PDU session", pduSessID)
		plan.outcomes = append(plan.outcomes, ngapTestpacket.QosFlowOutcome{QfiValue: qfi, Succeeded: true})
	}
	return plan
}

// qosParamsOrZero keeps the recorded flow usable when the request modified a flow without
// restating its parameters, which is permitted.
func qosParamsOrZero(p *ngapType.QosFlowLevelQosParameters) ngapType.QosFlowLevelQosParameters {
	if p == nil {
		return ngapType.QosFlowLevelQosParameters{}
	}
	return *p
}

// releasedQfis returns the QoS flows the request asks the radio to release.
func releasedQfis(transfer *ngapType.PDUSessionResourceModifyRequestTransfer) []int64 {
	var qfis []int64
	for _, ie := range transfer.ProtocolIEs.List {
		if ie.Id.Value != ngapType.ProtocolIEIDQosFlowToReleaseList || ie.Value.QosFlowToReleaseList == nil {
			continue
		}
		for _, item := range ie.Value.QosFlowToReleaseList.List {
			qfis = append(qfis, item.QosFlowIdentifier.Value)
		}
	}
	return qfis
}
