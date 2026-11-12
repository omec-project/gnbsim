// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/factory"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/gnodeb/ngap"
	"github.com/omec-project/gnbsim/gnodeb/worker/gnbupfworker"
	"github.com/omec-project/gnbsim/gnodeb/worker/gnbupueworker"
	"github.com/omec-project/gnbsim/ngapconv"
	"github.com/omec-project/gnbsim/stats"
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
	"github.com/omec-project/gnbsim/util/test"
	"github.com/omec-project/ngap/v2/aper"
	"github.com/omec-project/ngap/v2/ngapType"
)

type pduSessResourceSetupItem struct {
	NASPDU                                 *ngapType.NASPDU
	SNSSAI                                 ngapType.SNSSAI
	PDUSessionResourceSetupRequestTransfer aper.OctetString
	PDUSessionID                           ngapType.PDUSessionID
}

func HandleConnectRequest(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.UuMessage)
	gnbue.Supi = msg.Supi
	gnbue.WriteUeChan = msg.CommChan
}

func HandleInitialUEMessage(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.UuMessage)
	if gnbue.AmfUeNgapId != 0 {
		sendMsg, err := test.GetUplinkNASTransport(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, msg.NasPdus[0])
		if err != nil {
			gnbue.Log.Errorln("GetUplinkNASMessage failed:", err)
			return
		}
		err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg, msg.Id)
		if err != nil {
			gnbue.Log.Errorln("SendToPeer failed:", err)
			return
		}
		gnbue.Log.Debugln("sent Uplink NAS Message to AMF")
	} else {
		sendMsg, err := ngap.GetInitialUEMessage(gnbue, msg.NasPdus[0], msg.Tmsi)
		if err != nil {
			gnbue.Log.Errorln("GetInitialUEMessage failed:", err)
			return
		}
		err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg, msg.Id)
		if err != nil {
			gnbue.Log.Errorln("SendToPeer failed:", err)
			return
		}
		gnbue.Log.Debugln("sent Initial UE Message to AMF")
	}
}

func HandleDownlinkNasTransport(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.N2Message)
	// Need not perform other checks as they are validated at gnbamfworker level
	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdu *ngapType.NASPDU

	pdu := msg.NgapPdu

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

	// the decoder does not enforce mandatory IEs
	if amfUeNgapId == nil || nasPdu == nil {
		gnbue.Log.Errorln("DownlinkNASTransport is missing AMFUENGAPID or NASPDU")
		return
	}

	// TODO: check what needs to be done with AmfUeNgapId on every DownlinkNasTransport message
	gnbue.AmfUeNgapId = amfUeNgapId.Value
	var pdus common.NasPduList
	pdus = append(pdus, nasPdu.Value)
	SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, pdus, msg.Id)
}

func HandleUlInfoTransfer(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.UuMessage)
	gnbue.Log.Debugln("creating Uplink NAS Transport Message")
	sendMsg, err := ngap.GetUplinkNASTransport(gnbue, msg.NasPdus[0])
	if err != nil {
		gnbue.Log.Errorln("GetUplinkNASTransport failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg, msg.Id)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}

	gnbue.Log.Debugln("sent Uplink NAS Transport Message to AMF")
}

func HandleInitialContextSetupRequest(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.N2Message)
	var amfUeNgapId *ngapType.AMFUENGAPID
	var nasPdu *ngapType.NASPDU
	var pduSessResourceSetupReqList *ngapType.PDUSessionResourceSetupListCxtReq

	pdu := msg.NgapPdu

	initiatingMessage := pdu.InitiatingMessage
	initialContextSetupRequest := initiatingMessage.Value.InitialContextSetup

	for _, ie := range initialContextSetupRequest.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				gnbue.Log.Errorln("AMFUENGAPID is nil")
				return
			}
			gnbue.AmfUeNgapId = amfUeNgapId.Value
		case ngapType.ProtocolIEIDNASPDU:
			nasPdu = ie.Value.NASPDU
			if nasPdu == nil {
				gnbue.Log.Errorln("NASPDU is nil")
				return
			}
		case ngapType.ProtocolIEIDPDUSessionResourceSetupListCxtReq:
			pduSessResourceSetupReqList = ie.Value.PDUSessionResourceSetupListCxtReq
			if pduSessResourceSetupReqList == nil || len(pduSessResourceSetupReqList.List) == 0 {
				gnbue.Log.Errorln("PDUSessionResourceSetupListCxtReq is empty")
				return
			}
		}
	}

	// Said first, so the SimUe knows before any NAS message arrives that the procedure includes the
	// restoration of the user plane. The NAS-PDU is optional, so this cannot ride on it.
	if pduSessResourceSetupReqList != nil {
		// the NAS-PDU, if there is one, is the Service Accept: the SimUe is told whether to expect it
		var carried common.NasPduList
		if nasPdu != nil && nasPdu.Value != nil {
			carried = append(carried, nasPdu.Value)
		}
		pendingMsg := &common.UuMessage{
			DefaultMessage: common.DefaultMessage{Event: common.BEARER_SETUP_PENDING_EVENT},
			NasPdus:        carried,
			Id:             msg.Id,
			RanUeNgapId:    gnbue.GnbUeNgapId,
		}
		gnbue.Log.Debugln("sending event", pendingMsg.Event, "to SimUe. Id:", msg.Id)
		sendToSimUe(gnbue, pendingMsg)
	}

	// NAS-PDU is optional in the Initial Context Setup Request (TS 38.413 clause 9.2.2.1)
	if nasPdu != nil && nasPdu.Value != nil {
		var pdus common.NasPduList
		pdus = append(pdus, nasPdu.Value)
		SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, pdus, msg.Id)
		gnbue.Log.Debugln("sent DL Information Transfer Event to UE")
	}

	var list []pduSessResourceSetupItem
	if pduSessResourceSetupReqList != nil {
		for _, v := range pduSessResourceSetupReqList.List {
			dst := pduSessResourceSetupItem{}
			dst.NASPDU = v.NASPDU
			dst.PDUSessionID = v.PDUSessionID
			dst.SNSSAI = v.SNSSAI
			dst.PDUSessionResourceSetupRequestTransfer = v.PDUSessionResourceSetupRequestTransfer
			list = append(list, dst)
		}
	}

	if len(list) != 0 {
		ProcessPduSessResourceSetupList(gnbue, list, common.INITIAL_CTX_SETUP_REQUEST_EVENT, msg.Id)
		return
	}

	e := &stats.StatisticsEvent{Supi: gnbue.Supi, EType: stats.ICS_REQ_IN, Id: msg.Id}
	stats.LogStats(e)

	// TODO: Handle other mandatory IEs

	resp, err := test.GetInitialContextSetupResponse(gnbue.AmfUeNgapId, gnbue.GnbUeNgapId)
	if err != nil {
		gnbue.Log.Errorln("failed to create Initial Context Setup Response Message")
		return
	}

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, resp, 0)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}
}

// TODO: Error handling
func HandlePduSessResourceSetupRequest(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.N2Message)
	var amfUeNgapId *ngapType.AMFUENGAPID
	var pduSessResourceSetupReqList *ngapType.PDUSessionResourceSetupListSUReq

	pdu := msg.NgapPdu

	initiatingMessage := pdu.InitiatingMessage
	pduSessResourceSetupReq := initiatingMessage.Value.PDUSessionResourceSetup

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

	if pduSessResourceSetupReqList == nil {
		gnbue.Log.Errorln("PDUSessionResourceSetupListSUReq is missing")
		return
	}

	var list []pduSessResourceSetupItem
	for _, v := range pduSessResourceSetupReqList.List {
		dst := pduSessResourceSetupItem{}
		dst.NASPDU = v.PDUSessionNASPDU
		dst.PDUSessionID = v.PDUSessionID
		dst.SNSSAI = v.SNSSAI
		dst.PDUSessionResourceSetupRequestTransfer = v.PDUSessionResourceSetupRequestTransfer
		list = append(list, dst)
	}

	ProcessPduSessResourceSetupList(gnbue, list, common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT, msg.Id)
}

// TODO: Error handling
func HandlePduSessResourceReleaseCommand(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.N2Message)
	var amfUeNgapId *ngapType.AMFUENGAPID
	var pduSessResourceToReleaseList *ngapType.PDUSessionResourceToReleaseListRelCmd
	var nasPdu *ngapType.NASPDU

	pdu := msg.NgapPdu

	initiatingMessage := pdu.InitiatingMessage
	pduSessResourceReleaseCmd := initiatingMessage.Value.PDUSessionResourceRelease

	for _, ie := range pduSessResourceReleaseCmd.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID
			if amfUeNgapId == nil {
				gnbue.Log.Errorln("AMFUENGAPID is nil")
				return
			}
		case ngapType.ProtocolIEIDPDUSessionResourceToReleaseListRelCmd:
			pduSessResourceToReleaseList = ie.Value.PDUSessionResourceToReleaseListRelCmd
			if pduSessResourceToReleaseList == nil || len(pduSessResourceToReleaseList.List) == 0 {
				gnbue.Log.Errorln("PDUSessionResourceToReleaseListRelCmd is empty")
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

	if pduSessResourceToReleaseList == nil {
		gnbue.Log.Errorln("PDUSessionResourceToReleaseListRelCmd is missing")
		return
	}

	for _, item := range pduSessResourceToReleaseList.List {
		resourceReleaseCmdTransfer := ngapType.PDUSessionResourceReleaseCommandTransfer{}
		err := aper.UnmarshalWithParams(item.PDUSessionResourceReleaseCommandTransfer,
			&resourceReleaseCmdTransfer, "valueExt")
		if err != nil {
			gnbue.Log.Errorln("UnmarshalWithParams returned:", err)
			return
		}

		pduSessId := item.PDUSessionID.Value
		_, cause := test.PrintAndGetCause(&resourceReleaseCmdTransfer.Cause)
		gnbue.Log.Infoln("PDU Session Resource Release Command PDU Session ID:",
			pduSessId)
		gnbue.Log.Infoln("PDU Session Resource Release Command Cause:", cause)

		upCtx, err := gnbue.GetGnbUpUe(pduSessId)
		if err != nil {
			gnbue.Log.Errorln("failed to fetch PDU session context:", err)
			return
		}
		terminateUpUeContext(upCtx)
		gnbue.RemoveGnbUpUe(pduSessId)
	}

	// NAS-PDU is optional in the PDU Session Resource Release Command (TS 38.413 clause 9.2.1.3)
	if nasPdu != nil && nasPdu.Value != nil {
		var pdus common.NasPduList
		pdus = append(pdus, nasPdu.Value)
		SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, pdus, msg.Id)
		gnbue.Log.Debugln("sent DL Information Transfer Event to UE")
	}

	ngapPdu, err := test.GetPDUSessionResourceReleaseResponse(gnbue.AmfUeNgapId,
		gnbue.GnbUeNgapId)
	if err != nil {
		gnbue.Log.Errorln("failed to create PDU Session Resource Release Response:", err)
		return
	}

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, ngapPdu, 0)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}
	gnbue.Log.Debugln("sent PDU Session Resource Setup Response Message to AMF")

	SendToUe(gnbue, common.DATA_BEARER_RELEASE_REQUEST_EVENT, nil, msg.Id)
}

// HandleDataBearerSetupResponse answers the AMF for the sessions of a PDU Session Resource Setup
// Request, Initial Context Setup Request or Handover, and then tells the SimUe how that ended. The
// SimUe only completes the procedure on that report, since only the gNB knows whether the sessions
// really are in place and whether the AMF could be answered.
func HandleDataBearerSetupResponse(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.UuMessage)
	err := setupDataBearers(gnbue, msg)
	if err != nil {
		gnbue.Log.Errorln("data bearer setup failed:", err)
	}
	if gnbue.WriteUeChan == nil {
		return
	}
	result := &common.UuMessage{
		DefaultMessage:  common.DefaultMessage{Event: common.DATA_BEARER_SETUP_RESULT_EVENT, Error: err},
		Id:              msg.Id,
		TriggeringEvent: msg.TriggeringEvent,
		RanUeNgapId:     gnbue.GnbUeNgapId,
	}
	sendToSimUe(gnbue, result)
}

func setupDataBearers(gnbue *gnbctx.GnbCpUe, msg *common.UuMessage) error {
	var setupErr error
	var pduSessions []*ngapTestpacket.PduSession
	for _, item := range msg.DBParams {
		pduSess := item.PduSess
		if pduSess == nil {
			gnbue.Log.Errorln("data bearer params without PDU session, skipping")
			continue
		}
		if !pduSess.Success {
			// the context was created when the request was handled and no worker was started for it
			if upCtx, err := gnbue.GetGnbUpUe(pduSess.PduSessId); err == nil {
				releaseUnstartedUpUe(upCtx)
			}
			gnbue.RemoveGnbUpUe(pduSess.PduSessId)
		} else {
			gnbUpUe, err := gnbue.GetGnbUpUe(pduSess.PduSessId)
			if err != nil {
				gnbue.Log.Errorln("failed to fetch PDU session context:", err)
				// there is no user plane context to attach to, so the session cannot be reported as set up
				pduSess.Success = false
				pduSessions = append(pduSessions, pduSess)
				setupErr = fmt.Errorf("PDU session %d has no user plane context: %w", pduSess.PduSessId, err)
				continue
			}
			// TODO: Addition to this map should only be through GnbUpfWorker
			// routine. This will help in replacing sync map with normal map
			// Thus will help avoid lock unlock operation on per downlink message
			gnbUpUe.Upf.GnbUpUes.AddGnbUpUe(gnbUpUe.DlTeid, true, gnbUpUe)
			gnbUpUe.WriteUeChan = item.CommChan
			gnbUpUe.WorkerStarted.Store(true)
			gnbue.WaitGrp.Go(func() {
				gnbupueworker.Init(gnbUpUe)
			})
		}
		pduSessions = append(pduSessions, pduSess)
	}

	var ngapPdu []byte
	var err error

	switch msg.TriggeringEvent {
	case common.PDU_SESS_RESOURCE_SETUP_REQUEST_EVENT:
		ngapPdu, err = test.GetPDUSessionResourceSetupResponse(pduSessions,
			gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, gnbue.Gnb.GnbN3Ip)
		if err != nil {
			return fmt.Errorf("failed to create PDU Session Resource Setup Response: %w", err)
		}
	case common.INITIAL_CTX_SETUP_REQUEST_EVENT:
		ngapPdu, err = test.GetInitialContextSetupResponseForServiceRequest(pduSessions,
			gnbue.AmfUeNgapId, gnbue.GnbUeNgapId, gnbue.Gnb.GnbN3Ip)
		if err != nil {
			return fmt.Errorf("failed to create Initial Context Setup Response: %w", err)
		}
	case common.HO_NOTIFY_EVENT:
		// Target gNB: the user plane workers were started by the loop above, now that the RealUE
		// channels are wired up. Send HandoverNotify to the AMF.
		ngapPdu, err = ngap.GetHandoverNotify(gnbue)
		if err != nil {
			return fmt.Errorf("GetHandoverNotify failed: %w", err)
		}
	}

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, ngapPdu, msg.Id)
	if err != nil {
		return fmt.Errorf("SendToPeer failed: %w", err)
	}
	gnbue.Log.Debugln("sent PDU Session Resource Setup Response Message to AMF")
	return setupErr
}

func HandleUeCtxReleaseCommand(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	msg := intfcMsg.(*common.N2Message)
	var ueNgapIds *ngapType.UENGAPIDs
	var amfUeNgapId ngapType.AMFUENGAPID
	var cause *ngapType.Cause

	pdu := msg.NgapPdu

	initiatingMessage := pdu.InitiatingMessage
	ueCtxRelCmd := initiatingMessage.Value.UEContextRelease

	for _, ie := range ueCtxRelCmd.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDUENGAPIDs:
			ueNgapIds = ie.Value.UENGAPIDs
			if ueNgapIds == nil {
				gnbue.Log.Errorln("UENGAPIDs is nil")
				return
			}
		case ngapType.ProtocolIEIDCause:
			cause = ie.Value.Cause
			if cause == nil {
				gnbue.Log.Errorln("Cause is nil")
				return
			}
		}
	}

	if ueNgapIds == nil || cause == nil {
		gnbue.Log.Errorln("UEContextReleaseCommand is missing UENGAPIDs or Cause")
		return
	}

	_, causeNum := test.PrintAndGetCause(cause)

	if ueNgapIds.Present == ngapType.UENGAPIDsPresentUENGAPIDPair && ueNgapIds.UENGAPIDPair != nil {
		amfUeNgapId = ueNgapIds.UENGAPIDPair.AMFUENGAPID
		if gnbue.AmfUeNgapId != amfUeNgapId.Value {
			gnbue.Log.Errorln("AmfUeNgapId mismatch")
		}
	}

	ngapPdu, err := ngap.GetUEContextReleaseComplete(gnbue)
	if err != nil {
		gnbue.Log.Errorln("failed to create UE Context Release Complete message")
		return
	}

	e := &stats.StatisticsEvent{Supi: gnbue.Supi, EType: stats.UE_CTX_CMD_IN, Id: msg.Id}
	stats.LogStats(e)

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, ngapPdu, msg.Id)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}
	gnbue.Log.Debugln("sent UE Context Release Complete Message to AMF")

	quitEvt := &common.DefaultMessage{}
	quitEvt.Event = common.QUIT_EVENT
	gnbue.ReadChan <- quitEvt

	req := &common.UuMessage{}
	req.Event = common.CONNECTION_RELEASE_REQUEST_EVENT
	if causeNum == ngapType.CauseNasPresentDeregister {
		req.TriggeringEvent = common.DEREG_REQUEST_UE_ORIG_EVENT
	} else if cause.Present == ngapType.CausePresentRadioNetwork &&
		cause.RadioNetwork != nil &&
		cause.RadioNetwork.Value == ngapType.CauseRadioNetworkPresentSuccessfulHandover {
		// Released because of successful N2 handover: SimUe must not tear down
		// its connection to the target gNB, so mark with HO_COMMAND_EVENT.
		req.TriggeringEvent = common.HO_COMMAND_EVENT
	} else {
		req.TriggeringEvent = common.TRIGGER_AN_RELEASE_EVENT
	}

	sendToSimUe(gnbue, req)
}

func HandleRanConnectionRelease(gnbue *gnbctx.GnbCpUe,
	intfcMsg common.InterfaceMessage,
) {
	// Todo: The cause for the RAN connection release should be sent by the
	// Sim-UE, which should receive it through configuration
	gnbue.Log.Debugln("handling RAN Connection Release Event")

	gnbue.Log.Debugln("creating UE Context Release Request")

	sendMsg, err := ngap.GetUEContextReleaseRequest(gnbue)
	if err != nil {
		gnbue.Log.Errorln("GetUplinkNASTransport failed:", err)
		return
	}

	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: gnbue.Supi, EType: stats.UE_CTX_REL_OUT, Id: id}
	stats.LogStats(e)

	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg, id)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed:", err)
		return
	}

	gnbue.Log.Debugln("sent Uplink NAS Transport Message to AMF")
}

func ProcessPduSessResourceSetupList(gnbue *gnbctx.GnbCpUe,
	lst []pduSessResourceSetupItem, event common.EventType, id uint64,
) {
	// var pduSessions []ngapTestpacket.PduSession

	var dbParamSet []*common.DataBearerParams

	var nasPdus common.NasPduList

	for _, item := range lst {
		dbParam, err := setupPduSession(gnbue, item)
		if err != nil {
			// a session that cannot be set up is reported as failed (TS 38.413 clause 8.2.1.2)
			// without affecting the others, and its NAS PDU is not passed to the UE
			gnbue.Log.Errorf("failed to set up PDU session %d: %v", item.PDUSessionID.Value, err)
			dbParamSet = append(dbParamSet, &common.DataBearerParams{
				PduSess: &ngapTestpacket.PduSession{PduSessId: item.PDUSessionID.Value},
			})
			continue
		}
		if item.NASPDU != nil {
			nasPdus = append(nasPdus, item.NASPDU.Value)
		}
		dbParamSet = append(dbParamSet, dbParam)
	}

	if len(nasPdus) != 0 {
		SendToUe(gnbue, common.DL_INFO_TRANSFER_EVENT, nasPdus, id)
		gnbue.Log.Debugln("sent DL Information Transfer Event to UE")
	}

	/* TODO: To be fixed, currently Data Bearer Setup Event may get processed
	 * before the pdu sessions are established on the UE side
	 */
	time.Sleep(500 * time.Millisecond)
	uemsg := common.UuMessage{
		DefaultMessage:  common.DefaultMessage{Event: common.DATA_BEARER_SETUP_REQUEST_EVENT},
		DBParams:        dbParamSet,
		TriggeringEvent: event,
	}
	sendToSimUe(gnbue, &uemsg)
}

func HandleQuitEvent(gnbue *gnbctx.GnbCpUe, intfcMsg common.InterfaceMessage) {
	terminateUpUeContexts(gnbue)
	// unregister before the ID can be handed out again, so AMF messages for it find nobody
	gnbue.Gnb.GnbUes.RemoveGnbCpUe(gnbue.GnbUeNgapId)
	gnbue.Gnb.RanUeNGAPIDGenerator.FreeID(gnbue.GnbUeNgapId)
	gnbue.WaitGrp.Wait()
	gnbue.Log.Infoln("gNB Control-Plane UE context terminated")
}

// HandleTriggerHandover is called on the SOURCE gNB when SimUe initiates N2 handover.
// It sends a HandoverRequired message to the AMF.
func HandleTriggerHandover(gnbue *gnbctx.GnbCpUe, intfcMsg common.InterfaceMessage) {
	msg := intfcMsg.(*common.UuMessage)
	if msg.TargetGnbName == "" {
		gnbue.Log.Errorln("target gNB name is empty in TRIGGER_HO_EVENT")
		return
	}
	gnbue.Log.Infoln("handling Trigger Handover Event, target gNB:", msg.TargetGnbName)

	targetGnb, err := factory.AppConfig.Configuration.GetGNodeB(msg.TargetGnbName)
	if err != nil {
		gnbue.Log.Errorln("failed to get target gNB:", err)
		return
	}

	// The pre-registered target of this UE shares its SimUe channel. Its identity goes into the
	// container that the AMF forwards, so the target gNB can match the HandoverRequest to it.
	var container []byte
	if targetGnb.GnbUes != nil {
		if c := targetGnb.GnbUes.GetGnbCpUeByUeChan(gnbue.WriteUeChan); c != nil && c != gnbue {
			container = gnbctx.HandoverTargetToken(c.GnbUeNgapId, c.HandoverNonce)
		}
	}

	sendMsg, err := ngap.GetHandoverRequired(gnbue, targetGnb, container)
	if err != nil {
		gnbue.Log.Errorln("GetHandoverRequired failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, sendMsg, 0)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer failed for HandoverRequired:", err)
		return
	}
	gnbue.Log.Infoln("sent HandoverRequired to AMF")
}

// HandleHandoverRequest is called on the TARGET gNB when AMF sends HandoverRequest.
// It sets up the target-side resources and responds with HandoverRequestAcknowledge.
func HandleHandoverRequest(gnbue *gnbctx.GnbCpUe, intfcMsg common.InterfaceMessage) {
	msg := intfcMsg.(*common.N2Message)
	gnbue.Log.Infoln("handling Handover Request from AMF")

	pdu := msg.NgapPdu
	initiatingMessage := pdu.InitiatingMessage
	hoReq := initiatingMessage.Value.HandoverResourceAllocation

	var amfUeNgapId *ngapType.AMFUENGAPID
	var pduSessSetupList *ngapType.PDUSessionResourceSetupListHOReq

	for _, ie := range hoReq.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDAMFUENGAPID:
			amfUeNgapId = ie.Value.AMFUENGAPID
		case ngapType.ProtocolIEIDPDUSessionResourceSetupListHOReq:
			pduSessSetupList = ie.Value.PDUSessionResourceSetupListHOReq
		}
	}
	if amfUeNgapId == nil {
		gnbue.Log.Errorln("AMFUENGAPID is nil in HandoverRequest")
		return
	}
	gnbue.AmfUeNgapId = amfUeNgapId.Value

	var pduSessions []*ngapTestpacket.PduSession

	if pduSessSetupList != nil {
		for _, item := range pduSessSetupList.List {
			resourceSetupTransfer := ngapType.PDUSessionResourceSetupRequestTransfer{}
			if err := aper.UnmarshalWithParams(item.HandoverRequestTransfer,
				&resourceSetupTransfer, "valueExt"); err != nil {
				gnbue.Log.Errorln("UnmarshalWithParams HandoverRequest transfer failed:", err)
				continue
			}

			var gtpTunnel *ngapType.GTPTunnel
			var qosFlowSetupReqList *ngapType.QosFlowSetupRequestList
			for _, tfIE := range resourceSetupTransfer.ProtocolIEs.List {
				switch tfIE.Id.Value {
				case ngapType.ProtocolIEIDULNGUUPTNLInformation:
					gtpTunnel = tfIE.Value.ULNGUUPTNLInformation.GTPTunnel
				case ngapType.ProtocolIEIDQosFlowSetupRequestList:
					qosFlowSetupReqList = tfIE.Value.QosFlowSetupRequestList
				}
			}
			if gtpTunnel == nil {
				gnbue.Log.Warnln("no GTP tunnel found for PDU session, skipping")
				continue
			}
			if len(gtpTunnel.GTPTEID.Value) != 4 {
				gnbue.Log.Errorf("unexpected GTPTEID length %d, skipping PDU session", len(gtpTunnel.GTPTEID.Value))
				continue
			}
			ulteid := binary.BigEndian.Uint32(gtpTunnel.GTPTEID.Value)
			upfIp, _ := ngapconv.IPAddressToString(gtpTunnel.TransportLayerAddress)
			if upfIp == "" {
				gnbue.Log.Errorln("failed to resolve UPF IP address, skipping PDU session")
				continue
			}
			dlteid, err := gnbue.Gnb.DlTeidGenerator.Allocate()
			if err != nil {
				gnbue.Log.Errorln("DlTeid Allocate failed:", err)
				return
			}

			gnbupue := gnbctx.NewGnbUpUe(uint32(dlteid), ulteid, gnbue.Gnb)
			gnbupue.PduSessId = item.PDUSessionID.Value

			pduSess := &ngapTestpacket.PduSession{}
			pduSess.PduSessId = gnbupue.PduSessId
			pduSess.Teid = gnbupue.DlTeid
			pduSess.Success = true

			if qosFlowSetupReqList != nil {
				for _, qosItem := range qosFlowSetupReqList.List {
					pduSess.SuccessQfiList = append(pduSess.SuccessQfiList, qosItem.QosFlowIdentifier.Value)
					gnbupue.AddQosFlow(qosItem.QosFlowIdentifier.Value, &qosItem)
				}
			}

			gnbupf, created := gnbue.Gnb.GnbPeers.GetOrAddGnbUpf(upfIp)
			if created {
				go gnbupfworker.Init(gnbupf)
			}
			gnbupue.Upf = gnbupf
			gnbue.AddGnbUpUe(gnbupue.PduSessId, gnbupue)

			pduSessions = append(pduSessions, pduSess)
		}
	}

	// Send HandoverRequestAcknowledge to AMF
	ngapPdu, err := ngap.GetHandoverRequestAcknowledge(gnbue, pduSessions, gnbue.Gnb.GnbN3Ip)
	if err != nil {
		gnbue.Log.Errorln("GetHandoverRequestAcknowledge failed:", err)
		return
	}
	err = gnbue.Gnb.CpTransport.SendToPeer(gnbue.Amf, ngapPdu, msg.Id)
	if err != nil {
		gnbue.Log.Errorln("SendToPeer HandoverRequestAcknowledge failed:", err)
		return
	}
	gnbue.Log.Infoln("sent HandoverRequestAcknowledge to AMF")
}

// HandleHandoverCommand is called on the SOURCE gNB when AMF sends HandoverCommand.
// It forwards the event to SimUe so it can switch to the target gNB.
func HandleHandoverCommand(gnbue *gnbctx.GnbCpUe, intfcMsg common.InterfaceMessage) {
	msg := intfcMsg.(*common.N2Message)
	gnbue.Log.Infoln("handling Handover Command from AMF - forwarding to SimUe")
	// Forward to SimUe via the UuMessage channel so SimUe can switch gNBs
	uemsg := &common.UuMessage{}
	uemsg.Event = common.HO_COMMAND_EVENT
	uemsg.Id = msg.Id
	sendToSimUe(gnbue, uemsg)
}

// HandleHandoverNotify is called on the TARGET gNB to trigger the data bearer
// re-setup towards the RealUE.  The UP workers and the actual HandoverNotify
// NGAP message are sent after the RealUE acknowledges via
// DATA_BEARER_SETUP_RESPONSE_EVENT (handled by HandleDataBearerSetupResponse).
func HandleHandoverNotify(gnbue *gnbctx.GnbCpUe, intfcMsg common.InterfaceMessage) {
	gnbue.Log.Infoln("handling Handover Notify Event - setting up data bearers with RealUE")

	var dbParamSet []*common.DataBearerParams

	gnbue.GnbUpUes.Range(func(k, v any) bool {
		gnbUpUe := v.(*gnbctx.GnbUpUe)
		pduSess := &ngapTestpacket.PduSession{}
		pduSess.PduSessId = gnbUpUe.PduSessId
		pduSess.Teid = gnbUpUe.DlTeid
		pduSess.Success = true

		dbParam := &common.DataBearerParams{}
		dbParam.CommChan = gnbUpUe.ReadUlChan
		dbParam.PduSess = pduSess
		dbParamSet = append(dbParamSet, dbParam)
		return true
	})

	// Ask RealUE to update its pdusessworker channels to the target gNB.
	// TriggeringEvent = HO_NOTIFY_EVENT distinguishes this from a normal PDU
	// session setup response in HandleDataBearerSetupResponse.
	// TODO: replace sleep with proper synchronization once the ICS race (line 651) is resolved.
	time.Sleep(500 * time.Millisecond)
	uemsg := common.UuMessage{
		DefaultMessage:  common.DefaultMessage{Event: common.DATA_BEARER_SETUP_REQUEST_EVENT},
		DBParams:        dbParamSet,
		TriggeringEvent: common.HO_NOTIFY_EVENT,
	}
	sendToSimUe(gnbue, &uemsg)
}

func terminateUpUeContexts(gnbue *gnbctx.GnbCpUe) {
	f := func(key, value any) bool {
		terminateUpUeContext(value.(*gnbctx.GnbUpUe))
		return true
	}
	gnbue.GnbUpUes.Range(f)
	gnbue.GnbUpUes = sync.Map{}
}

// releaseUnstartedUpUe frees a user plane context whose worker was never started: its downlink
// tunnel ID is otherwise only freed by the worker's quit, which nothing would run.
func releaseUnstartedUpUe(upCtx *gnbctx.GnbUpUe) {
	if upCtx.WorkerStarted.Load() {
		return
	}
	if upCtx.Gnb != nil && upCtx.Gnb.DlTeidGenerator != nil {
		upCtx.Gnb.DlTeidGenerator.FreeID(int64(upCtx.DlTeid))
	}
	if upCtx.Upf != nil {
		upCtx.Upf.GnbUpUes.RemoveGnbUpUe(upCtx.DlTeid, true)
	}
}

func terminateUpUeContext(upCtx *gnbctx.GnbUpUe) {
	if !upCtx.WorkerStarted.Load() {
		releaseUnstartedUpUe(upCtx)
		return
	}
	msg := &common.DefaultMessage{}
	msg.Event = common.QUIT_EVENT
	upCtx.RequestQuit() // frees a worker that is blocked sending to the UE, so it reads the command
	upCtx.ReadCmdChan <- msg
	upCtx.Upf.GnbUpUes.RemoveGnbUpUe(upCtx.DlTeid, true)
}

// setupPduSession creates the user plane context of one item of a PDU Session Resource Setup
// Request or Initial Context Setup Request. On error nothing is left allocated, so the caller can
// report the session as failed and carry on with the others.
func setupPduSession(gnbue *gnbctx.GnbCpUe, item pduSessResourceSetupItem) (dbParam *common.DataBearerParams, err error) {
	resourceSetupRequestTransfer := ngapType.PDUSessionResourceSetupRequestTransfer{}
	err = aper.UnmarshalWithParams(item.PDUSessionResourceSetupRequestTransfer,
		&resourceSetupRequestTransfer, "valueExt")
	if err != nil {
		return nil, fmt.Errorf("UnmarshalWithParams returned: %w", err)
	}

	var gtpTunnel *ngapType.GTPTunnel
	var pduSessType *ngapType.PDUSessionType
	var qosFlowSetupReqList *ngapType.QosFlowSetupRequestList
	for _, ie := range resourceSetupRequestTransfer.ProtocolIEs.List {
		switch ie.Id.Value {
		case ngapType.ProtocolIEIDULNGUUPTNLInformation:
			gtpTunnel = ie.Value.ULNGUUPTNLInformation.GTPTunnel
			if gtpTunnel == nil {
				return nil, errors.New("GTPTunnel is nil")
			}
		case ngapType.ProtocolIEIDPDUSessionType:
			pduSessType = ie.Value.PDUSessionType
			if pduSessType == nil {
				return nil, errors.New("PDUSessionType is nil")
			}
		case ngapType.ProtocolIEIDQosFlowSetupRequestList:
			qosFlowSetupReqList = ie.Value.QosFlowSetupRequestList
			if qosFlowSetupReqList == nil || len(qosFlowSetupReqList.List) == 0 {
				return nil, errors.New("QosFlowSetupRequestList is empty")
			}
		}
	}

	// the decoder does not enforce mandatory IEs
	if gtpTunnel == nil || pduSessType == nil || qosFlowSetupReqList == nil {
		return nil, errors.New("PDUSessionResourceSetupRequestTransfer is missing a mandatory IE")
	}
	if len(gtpTunnel.GTPTEID.Value) != 4 {
		return nil, fmt.Errorf("unexpected GTPTEID length %d", len(gtpTunnel.GTPTEID.Value))
	}
	upfIp, _ := ngapconv.IPAddressToString(gtpTunnel.TransportLayerAddress)
	if upfIp == "" {
		return nil, errors.New("failed to resolve UPF IP address")
	}
	ulteid := binary.BigEndian.Uint32(gtpTunnel.GTPTEID.Value)
	dlteid, err := gnbue.Gnb.DlTeidGenerator.Allocate()
	if err != nil {
		return nil, fmt.Errorf("ID Generator Allocate() returned: %w", err)
	}
	defer func() {
		if err != nil {
			gnbue.Gnb.DlTeidGenerator.FreeID(dlteid)
		}
	}()

	gnbupue := gnbctx.NewGnbUpUe(uint32(dlteid), ulteid, gnbue.Gnb)
	snssai, err := ngapconv.SNssaiToModels(item.SNSSAI)
	if err != nil {
		return nil, fmt.Errorf("SNssaiToModels returned: %w", err)
	}
	gnbupue.Snssai = snssai
	gnbupue.PduSessId = item.PDUSessionID.Value
	gnbupue.PduSessType = test.PDUSessionTypeToModels(*pduSessType)
	pduSess := &ngapTestpacket.PduSession{}
	pduSess.PduSessId = gnbupue.PduSessId
	pduSess.Teid = gnbupue.DlTeid

	gnbue.Log.Infoln("PDU Session ID:", gnbupue.PduSessId)
	gnbue.Log.Infoln("S-NSSAI - SST: ", gnbupue.Snssai.GetSst())
	gnbue.Log.Infoln("S-NSSAI - SD: ", gnbupue.Snssai.GetSd())
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
				return nil, errors.New("NonDynamic5QI is nil")
			}
			gnbue.Log.Infoln("Non Dynamic 5QI:", nonDynamic5QI.FiveQI.Value)
		}
		gnbue.Log.Infoln("ARP Priority Level:", arp.PriorityLevelARP.Value)
		gnbue.Log.Infoln("Pre-emption Capability:", arp.PreEmptionCapability.Value)
		gnbue.Log.Infoln("Pre-emption Vulnerability:", arp.PreEmptionVulnerability.Value)

		pduSess.SuccessQfiList = append(pduSess.SuccessQfiList, qosFlowId)
		gnbupue.AddQosFlow(qosFlowId, &qosFlowSetupReqItem)
	}

	pduSess.Success = true

	gnbupf, created := gnbue.Gnb.GnbPeers.GetOrAddGnbUpf(upfIp)
	if created {
		go gnbupfworker.Init(gnbupf)
	}
	gnbupue.Upf = gnbupf
	gnbue.AddGnbUpUe(gnbupue.PduSessId, gnbupue)

	dbParam = &common.DataBearerParams{}
	dbParam.CommChan = gnbupue.ReadUlChan
	dbParam.PduSess = pduSess
	return dbParam, nil
}
