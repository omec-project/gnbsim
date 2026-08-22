// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package realue

import (
	"fmt"
	"net"
	"strconv"

	"github.com/omec-project/gnbsim/common"
	realuectx "github.com/omec-project/gnbsim/realue/context"
	realue_nas "github.com/omec-project/gnbsim/realue/nas"
	"github.com/omec-project/gnbsim/realue/util"
	"github.com/omec-project/gnbsim/realue/worker/pdusessworker"
	"github.com/omec-project/gnbsim/stats"
	"github.com/omec-project/nas/v2"
	"github.com/omec-project/nas/v2/nasConvert"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/nasTestpacket"
	"github.com/omec-project/nas/v2/nasType"
	"github.com/omec-project/openapi/v2/models"
)

// TODO Remove the hardcoding
const (
	SWITCH_OFF                     uint8 = 0
	REQUEST_TYPE_EXISTING_PDU_SESS uint8 = 0x02
	// modificationRequestPTI is the identity the UE puts on its modification requests. One is
	// enough while only one such procedure runs at a time, and it is non-zero, which is what
	// distinguishes a UE-requested procedure from a network-requested one.
	modificationRequestPTI uint8 = 0x01
)

func HandleRegRequestEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_REQ_OUT, Id: id}
	stats.LogStats(e)

	ueSecurityCapability := ue.GetUESecurityCapability()

	ue.Suci, err = util.SupiToSuci(ue.Supi, ue.Plmn)
	if err != nil {
		ue.Log.Errorln("SupiToSuci returned:", err)
		return fmt.Errorf("failed to derive suci")
	}
	mobileId5GS := nasType.MobileIdentity5GS{
		Len:    uint16(len(ue.Suci)), // suci
		Buffer: ue.Suci,
	}
	ue.Log.Debugln("generating SUPI Registration Request Message")
	nasPdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileId5GS, nil, ueSecurityCapability, nil, nil, nil)

	m := formUuMessage(common.REG_REQUEST_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Debugln("sent Registration Request Message to SimUe")
	return nil
}

func HandleAuthResponseEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)

	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.AUTH_RSP_OUT, Id: id}
	stats.LogStats(e)
	msg.Id = id

	// First process the corresponding Auth Request
	ue.Log.Debugln("processing corresponding Authentication Request Message")
	authReq := msg.NasMsg.AuthenticationRequest

	ue.NgKsi = nasConvert.SpareHalfOctetAndNgksiToModels(authReq.SpareHalfOctetAndNgksi)

	mcc, err := strconv.Atoi(ue.Plmn.Mcc)
	if err != nil {
		ue.Log.Infoln("failed to convert mcc to int", err)
	}
	mnc, err := strconv.Atoi(ue.Plmn.Mnc)
	if err != nil {
		ue.Log.Infoln("failed to convert mnc to int", err)
	}
	snName := fmt.Sprintf("5G:mnc%03d.mcc%03d.3gppnetwork.org", mnc, mcc)

	rand := authReq.GetRANDValue()
	autn := authReq.GetAUTN()
	resStat := ue.DeriveRESstarAndSetKey(autn[:], rand[:], snName)

	// TODO: Parse Auth Request IEs and update the RealUE Context

	// Now generate NAS Authentication Response
	ue.Log.Debugln("generating Authentication Response Message")
	nasPdu := nasTestpacket.GetAuthenticationResponse(resStat, "")

	m := formUuMessage(common.AUTH_RESPONSE_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Debugln("sent Authentication Response Message to SimUe")
	return nil
}

func HandleSecModCompleteEvent(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) (err error) {
	// TODO: Process corresponding Security Mode Command first

	mobileId5GS := nasType.MobileIdentity5GS{
		Len:    uint16(len(ue.Suci)), // suci
		Buffer: ue.Suci,
	}
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileId5GS, nil,
		ue.GetUESecurityCapability(), ue.Get5GMMCapability(), nil, nil)

	ue.Log.Debugln("generating Security Mode Complete Message")
	nasPdu := nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext,
		true)
	if err != nil {
		ue.Log.Errorln("EncodeNasPduWithSecurity() returned:", err)
		return fmt.Errorf("failed to encrypt security mode complete message")
	}

	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.SECM_CMP_OUT, Id: id}
	stats.LogStats(e)

	m := formUuMessage(common.SEC_MOD_COMPLETE_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Debugln("sent Security Mode Complete Message to SimUe")
	return nil
}

func HandleRegCompleteEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	// TODO: Process corresponding Registration Accept first
	msg := intfcMsg.(*common.UeMessage).NasMsg.RegistrationAccept

	var guti []uint8
	if msg.GUTI5G != nil {
		guti = msg.GUTI5G.Octet[:]
	}

	_, ue.Guti = nasConvert.GutiToString(guti)

	ue.Log.Debugln("generating Registration Complete Message")
	nasPdu := nasTestpacket.GetRegistrationComplete(nil)
	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		ue.Log.Errorln("EncodeNasPduWithSecurity() returned:", err)
		return fmt.Errorf("failed to encrypt registration complete message")
	}

	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.REG_COMP_OUT, Id: id}
	stats.LogStats(e)

	m := formUuMessage(common.REG_COMPLETE_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Debugln("sent Registration Complete Message to SimUe")
	return nil
}

func HandleDeregRequestEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	if ue.Guti == "" {
		ue.Log.Errorln("guti not allocated")
		return fmt.Errorf("failed to create deregistration request: guti not unallocated")
	}
	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.DEREG_REQ_OUT, Id: id}
	stats.LogStats(e)

	gutiNas := nasConvert.GutiToNas(ue.Guti)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    11, // 5g-guti
		Buffer: gutiNas.Octet[:],
	}

	nasPdu := nasTestpacket.GetDeregistrationRequest(nasMessage.AccessType3GPP,
		SWITCH_OFF, uint8(ue.NgKsi.Ksi), mobileIdentity5GS)
	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		ue.Log.Errorln("EncodeNasPduWithSecurity() returned:", err)
		return fmt.Errorf("failed to encrypt deregistration request message")
	}

	m := formUuMessage(common.DEREG_REQUEST_UE_ORIG_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Debugln("sent UE Initiated Deregistration Request message to SimUe")
	return nil
}

func HandlePduSessEstRequestEvent(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) (err error) {
	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.PDU_SESS_REQ_OUT, Id: id}
	stats.LogStats(e)

	// sNssai := models.Snssai{
	// 	Sst: 1,
	// 	Sd:  "010203",
	// }
	nasPdu := nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10,
		nasMessage.ULNASTransportRequestTypeInitialRequest, ue.Dnn, ue.SNssai)

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		ue.Log.Errorln("failed to encrypt PDU Session Establishment Request Message", err)
		return
	}

	m := formUuMessage(common.PDU_SESS_EST_REQUEST_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	return nil
}

func HandlePduSessEstAcceptEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	nasMsg := msg.NasMsg.PDUSessionEstablishmentAccept
	if nasMsg == nil {
		ue.Log.Errorln("PDUSessionEstablishmentAccept is nil")
		return fmt.Errorf("invalid NAS Message")
	}

	var pduAddr net.IP
	pduSessType := nasConvert.PDUSessionTypeToModels(nasMsg.GetPDUSessionType())
	if pduSessType == models.PDUSESSIONTYPE_IPV4 {
		ip := nasMsg.GetPDUAddressInformation()
		pduAddr = net.IPv4(ip[0], ip[1], ip[2], ip[3])
	}

	pduSess := realuectx.NewPduSession(ue, int64(nasMsg.PDUSessionID.Octet))
	pduSess.PduSessType = pduSessType
	pduSess.SscMode = nasMsg.GetSSCMode()
	pduSess.PduAddress = pduAddr
	pduSess.WriteUeChan = ue.ReadChan
	ue.AddPduSession(pduSess.PduSessId, pduSess)
	ue.Log.Infoln("PDU Session ID:", pduSess.PduSessId)
	ue.Log.Infoln("PDU Session Type:", pduSess.PduSessType)
	ue.Log.Infoln("SSC Mode:", pduSess.SscMode)
	ue.Log.Infoln("PDU Address:", pduAddr.String())

	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.PDU_SESS_ACC_IN, Id: msg.Id}
	stats.LogStats(e)

	return nil
}

func HandlePduSessReleaseRequestEvent(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) (err error) {
	nasPdu := nasTestpacket.GetUlNasTransport_PduSessionReleaseRequest(10)

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		ue.Log.Errorln("failed to encrypt PDU Session Release Request Message", err)
		return
	}

	m := formUuMessage(common.PDU_SESS_REL_REQUEST_EVENT, nasPdu, 0)
	SendToSimUe(ue, m)
	return nil
}

func HandlePduSessReleaseCompleteEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	nasMsg := msg.NasMsg.PDUSessionReleaseCommand
	if nasMsg == nil {
		ue.Log.Errorln("PDUSessionReleaseCommand is nil")
		return fmt.Errorf("invalid NAS Message")
	}

	pduSessId := nasMsg.PDUSessionID.Octet
	ue.Log.Infoln("PDU Session Release Command, PDU Session ID:", pduSessId)

	pduSess, err := ue.GetPduSession(int64(pduSessId))
	if err != nil {
		return fmt.Errorf("failed to fetch PDU session:%v", err)
	}

	quitMsg := &common.UeMessage{}
	quitMsg.Event = common.QUIT_EVENT
	pduSess.ReadCmdChan <- quitMsg

	nasPdu := nasTestpacket.GetUlNasTransport_PduSessionReleaseComplete(pduSessId,
		REQUEST_TYPE_EXISTING_PDU_SESS, "", nil)

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		return fmt.Errorf("failed to encrypt PDU Session Release Request Message: %v", err)
	}

	m := formUuMessage(common.PDU_SESS_REL_COMPLETE_EVENT, nasPdu, 0)
	SendToSimUe(ue, m)
	return nil
}

// HandlePduSessModificationCompleteEvent answers a PDU SESSION MODIFICATION COMMAND.
//
// The UE takes the authorized parameters as given. TS 24.501 subclause 6.3.2.3: the command
// carries what the network has decided, not a proposal, so the answer confirms rather than
// negotiates. What the UE records is therefore the authorized QoS rules and flow descriptions as
// received, and the acknowledgement echoes the command's PTI so the network can match it.
// HandlePduSessModificationRequestEvent asks the network to modify a session.
//
// This exists to verify the network's refusal, not to obtain a modification: the core declines
// every UE-requested modification with a 5GSM cause. What matters is that the request is
// well-formed, that the PTI is the UE's own so the answer can be matched to it, and that the
// Request type IE carries whatever the profile asked for — including the wrong value, which is
// the case worth testing.
func HandlePduSessModificationRequestEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	_ = intfcMsg

	// The simulator runs one PDU session per UE, established as session 10, and the request
	// concerns that one. Taking it from the context rather than assuming the number keeps this
	// honest if that ever changes.
	pduSessID, pduSess, err := ue.OnlyPduSession()
	if err != nil {
		return fmt.Errorf("cannot request a modification: %v", err)
	}

	// Any non-zero value identifies the procedure; zero is reserved for network-requested ones.
	pduSess.PendingPTI = modificationRequestPTI

	requestType := ue.ModificationRequestType
	if requestType == 0 && !ue.OmitModificationRequestType {
		requestType = nasMessage.ULNASTransportRequestTypeModificationRequest
	}

	nasPdu, err := realue_nas.GetUlNasTransportPduSessionModificationRequest(
		uint8(pduSessID), pduSess.PendingPTI, requestType)
	if err != nil {
		return fmt.Errorf("failed to build PDU Session Modification Request: %v", err)
	}

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		return fmt.Errorf("failed to encrypt PDU Session Modification Request: %v", err)
	}

	ue.Log.Infof("sending PDU session modification request for session %d, PTI %d, request type %d",
		pduSessID, pduSess.PendingPTI, requestType)

	m := formUuMessage(common.UL_INFO_TRANSFER_EVENT, nasPdu, 0)
	SendToSimUe(ue, m)
	return nil
}

// HandlePduSessModificationRejectEvent records the network's refusal of a UE-requested
// modification.
//
// The refusal is the expected outcome, so this is not a failure path. What is checked is that the
// answer belongs to the request: a reject carrying a different PTI cannot be matched to the
// procedure the UE started, and a UE that accepted it anyway would clear the wrong procedure.
func HandlePduSessModificationRejectEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	nasMsg := msg.NasMsg.PDUSessionModificationReject
	if nasMsg == nil {
		return fmt.Errorf("PDUSessionModificationReject is nil")
	}

	pduSessId := nasMsg.PDUSessionID.Octet
	pti := nasMsg.PTI.Octet
	cause := nasMsg.Cause5GSM.Octet

	ue.Log.Infof("PDU session modification refused: session %d, PTI %d, 5GSM cause #%d",
		pduSessId, pti, cause)

	if pduSess, sessErr := ue.GetPduSession(int64(pduSessId)); sessErr == nil {
		if pduSess.PendingPTI != 0 && pti != pduSess.PendingPTI {
			ue.Log.Errorf("reject carries PTI %d but this UE's outstanding request used PTI %d; it cannot be matched to the procedure",
				pti, pduSess.PendingPTI)
		}
		pduSess.PendingPTI = 0
	}

	return nil
}

func HandlePduSessModificationCompleteEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UeMessage)
	nasMsg := msg.NasMsg.PDUSessionModificationCommand
	if nasMsg == nil {
		ue.Log.Errorln("PDUSessionModificationCommand is nil")
		return fmt.Errorf("invalid NAS Message")
	}

	pduSessId := nasMsg.PDUSessionID.Octet
	pti := nasMsg.PTI.Octet
	ue.Log.Infoln("PDU Session Modification Command, PDU Session ID:", pduSessId, "PTI:", pti)

	// The session must exist. A command for one that does not is not something to answer with a
	// complete — the network and the UE disagree about what exists, and confirming would hide it.
	if _, sessErr := ue.GetPduSession(int64(pduSessId)); sessErr != nil {
		return fmt.Errorf("modification command for an unknown PDU session %d: %v", pduSessId, sessErr)
	}

	if nasMsg.AuthorizedQosRules != nil {
		ue.Log.Infoln("authorized QoS rules received, length:", nasMsg.AuthorizedQosRules.GetLen())
	}
	if nasMsg.AuthorizedQosFlowDescriptions != nil {
		ue.Log.Infoln("authorized QoS flow descriptions received, length:",
			nasMsg.AuthorizedQosFlowDescriptions.GetLen())
	}
	if nasMsg.SessionAMBR != nil {
		ue.Log.Infoln("session AMBR received in the modification command")
	}

	nasPdu, err := realue_nas.GetUlNasTransportPduSessionModificationComplete(pduSessId, pti)
	if err != nil {
		return fmt.Errorf("failed to build PDU Session Modification Complete: %v", err)
	}

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true)
	if err != nil {
		return fmt.Errorf("failed to encrypt PDU Session Modification Complete: %v", err)
	}

	m := formUuMessage(common.PDU_SESS_MOD_COMPLETE_EVENT, nasPdu, 0)
	SendToSimUe(ue, m)
	return nil
}

func HandleDataBearerSetupRequestEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	for _, item := range msg.DBParams {
		/* Currently gNB also adds failed pdu session ids in the list.
		   pdu sessions are marked failed during decoding. real ue simply
		   returns the same list back by marking any failed pdu sessions on
		   its side. This consolidated list can be used by gnb to form
		   PDUSession Resource Setup/failed To Setup Response list
		*/
		if item.PduSess.Success {
			pduSess, err := ue.GetPduSession(item.PduSess.PduSessId)
			if err != nil {
				ue.Log.Warnln("failed to fetch PDU Session:", err)
				item.PduSess.Success = false
				continue
			}

			if !pduSess.Launched {
				pduSess.Launched = true
				ue.WaitGrp.Add(1)
				go pdusessworker.Init(pduSess, &ue.WaitGrp)
			}

			initMsg := &common.UeMessage{}
			initMsg.Event = common.INIT_EVENT
			initMsg.CommChan = item.CommChan
			pduSess.ReadCmdChan <- initMsg

			/* gNb can use this channel to send DL packets for this PDU session */
			item.CommChan = pduSess.ReadDlChan
		}
	}

	rsp := &common.UuMessage{}
	rsp.Event = common.DATA_BEARER_SETUP_RESPONSE_EVENT
	rsp.DBParams = msg.DBParams
	rsp.TriggeringEvent = msg.TriggeringEvent
	rsp.Id = stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.PDU_SESS_RES_SETUP, Id: rsp.Id}
	stats.LogStats(e)
	ue.WriteSimUeChan <- rsp
	return nil
}

func HandleDataPktGenRequestEvent(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) (err error) {
	for _, v := range ue.PduSessions {
		v.ReadCmdChan <- msg
	}

	return nil
}

func HandleDataPktGenSuccessEvent(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) (err error) {
	ue.WriteSimUeChan <- msg
	return nil
}

func HandleConnectionReleaseRequestEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)

	for _, pdusess := range ue.PduSessions {
		pdusess.ReadCmdChan <- msg
	}

	return nil
}

func HandleErrorEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	SendToSimUe(ue, intfcMsg)
	return nil
}

func HandleQuitEvent(ue *realuectx.RealUe, intfcMsg common.InterfaceMessage) (err error) {
	ue.WriteSimUeChan = nil
	for _, pdusess := range ue.PduSessions {
		pdusess.ReadCmdChan <- intfcMsg
	}
	ue.PduSessions = nil
	ue.WaitGrp.Wait()
	ue.Log.Infoln("Real UE terminated")
	return nil
}

func HandleDlInfoTransferEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	for _, pdu := range msg.NasPdus {
		nasMsg, err := realue_nas.NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
		if err != nil {
			ue.Log.Errorln("failed to decode dowlink NAS Message due to", err)
			return err
		}
		msgType := nasMsg.GmmHeader.GetMessageType()
		ue.Log.Infoln("Received Message Type:", msgType)

		if msgType == nas.MsgTypeDLNASTransport {
			ue.Log.Info("Payload contaner type:",
				nasMsg.DLNASTransport.SpareHalfOctetAndPayloadContainerType)
			payload := nasMsg.DLNASTransport.PayloadContainer
			if payload.Len == 0 {
				return fmt.Errorf("payload container length is 0")
			}
			buffer := payload.Buffer[:payload.Len]
			m := nas.NewMessage()
			err := m.PlainNasDecode(&buffer)
			if err != nil {
				ue.Log.Errorln("PlainNasDecode returned:", err)
				return fmt.Errorf("failed to decode payload container")
			}
			nasMsg = m
			msgType = nasMsg.GsmHeader.GetMessageType()
		}

		m := &common.UeMessage{}

		// The MSB out of the 32 bytes represents event type, which in this case
		// is N1_EVENT
		m.Event = common.EventType(msgType) | common.N1_EVENT
		m.NasMsg = nasMsg
		m.Id = msg.Id

		// Simply notify SimUe about the received nas message. Later SimUe will
		// asynchrously send next event to RealUE informing about what to do with
		// the received NAS message
		SendToSimUe(ue, m)
	}
	return nil
}

func HandleServiceRequestEvent(ue *realuectx.RealUe,
	msg common.InterfaceMessage,
) (err error) {
	nasPdu, err := realue_nas.GetServiceRequest(ue)
	if err != nil {
		return fmt.Errorf("failed to handle service request event: %v", err)
	}

	id := stats.GetId()
	e := &stats.StatisticsEvent{Supi: ue.Supi, EType: stats.SVC_REQ_OUT, Id: id}
	stats.LogStats(e)

	// TS 24.501 Section 4.4.6 - Protection of Initial NAS signalling messages
	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtected, true)
	if err != nil {
		return fmt.Errorf("failed to encode with security: %v", err)
	}

	m := formUuMessage(common.SERVICE_REQUEST_EVENT, nasPdu, id)
	var tmsi string
	if len(ue.Guti) == 19 {
		tmsi = ue.Guti[5:]
	} else {
		tmsi = ue.Guti[6:]
	}

	m.Tmsi = tmsi
	SendToSimUe(ue, m)
	return nil
}

func HandleNwDeregAcceptEvent(ue *realuectx.RealUe, msg common.InterfaceMessage) (err error) {
	ue.Log.Debugln("generating Dereg Accept Message")
	nasPdu := nasTestpacket.GetDeregistrationAccept()

	nasPdu, err = realue_nas.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext,
		true)
	if err != nil {
		ue.Log.Errorln("EncodeNasPduWithSecurity() returned:", err)
		return fmt.Errorf("failed to encrypt security mode complete message")
	}

	m := formUuMessage(common.DEREG_ACCEPT_UE_TERM_EVENT, nasPdu, 0)
	SendToSimUe(ue, m)
	ue.Log.Debugln("sent Dereg Accept UE Terminated Message to SimUe")
	return nil
}
