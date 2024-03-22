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
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasConvert"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/openapi/models"
)

// TODO Remove the hardcoding
const (
	SWITCH_OFF                     uint8 = 0
	REQUEST_TYPE_EXISTING_PDU_SESS uint8 = 0x02
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
	ue.Log.Traceln("Generating SUPI Registration Request Message")
	nasPdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileId5GS, nil, ueSecurityCapability, nil, nil, nil)

	m := formUuMessage(common.REG_REQUEST_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Traceln("Sent Registration Request Message to SimUe")
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
	ue.Log.Traceln("Processing corresponding Authentication Request Message")
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
	ue.Log.Traceln("Generating Authentication Response Message")
	nasPdu := nasTestpacket.GetAuthenticationResponse(resStat, "")

	m := formUuMessage(common.AUTH_RESPONSE_EVENT, nasPdu, id)
	SendToSimUe(ue, m)
	ue.Log.Traceln("Sent Authentication Response Message to SimUe")
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

	ue.Log.Traceln("Generating Security Mode Complete Message")
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
	ue.Log.Traceln("Sent Security Mode Complete Message to SimUe")
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

	ue.Log.Traceln("Generating Registration Complete Message")
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
	ue.Log.Traceln("Sent Registration Complete Message to SimUe")
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
	ue.Log.Traceln("Sent UE Initiated Deregistration Request message to SimUe")
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
		fmt.Println("Failed to encrypt PDU Session Establishment Request Message", err)
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
	if pduSessType == models.PduSessionType_IPV4 {
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
		fmt.Println("Failed to encrypt PDU Session Release Request Message", err)
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

func HandleDataBearerSetupRequestEvent(ue *realuectx.RealUe,
	intfcMsg common.InterfaceMessage,
) (err error) {
	msg := intfcMsg.(*common.UuMessage)
	for _, item := range msg.DBParams {
		/* Currently gNB also adds failed pdu session ids in the list.
		   pdu sessions are marked failed during decoding. real ue simply
		   returns the same list back by marking any failed pdu sessions on
		   its side. This consolidated list can be used by gnb to form
		   PDUSession Resource Setup/Failed To Setup Response list
		*/
		if item.PduSess.Success {
			pduSess, err := ue.GetPduSession(item.PduSess.PduSessId)
			if err != nil {
				ue.Log.Warnln("Failed to fetch PDU Session:", err)
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
			ue.Log.Errorln("Failed to decode dowlink NAS Message due to", err)
			return err
		}
		msgType := nasMsg.GmmHeader.GetMessageType()
		ue.Log.Infoln("Received Message Type:", msgType)

		if msgType == nas.MsgTypeDLNASTransport {
			ue.Log.Info("Payload contaner type:",
				nasMsg.GmmMessage.DLNASTransport.SpareHalfOctetAndPayloadContainerType)
			payload := nasMsg.GmmMessage.DLNASTransport.PayloadContainer
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
	ue.Log.Traceln("Generating Dereg Accept Message")
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
	ue.Log.Traceln("Sent Dereg Accept UE Terminated Message to SimUe")
	return nil
}
