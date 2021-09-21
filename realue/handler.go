// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package realue

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/realue/context"
	"gnbsim/realue/util"
	"gnbsim/util/test"

	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasTestpacket"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/openapi/models"
	"github.com/omec-project/nas"
)

//TODO Remove the hardcoding
var snName string = "5G:mnc093.mcc208.3gppnetwork.org"

func HandleRegReqEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Registration Request Event")

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

	ue.Log.Traceln("Generating Registration Request Message")
	nasPdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileId5GS, nil, ueSecurityCapability, nil, nil, nil)

	SendToSimUe(ue, common.REG_REQUEST_EVENT, nasPdu, nil)
	ue.Log.Traceln("Sent Registration Request Message to SimUe")
	return nil
}

func HandleAuthResponseEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Authentication Response Event")

	// First process the corresponding Auth Request
	ue.Log.Traceln("Processing corresponding Authentication Request Message")
	nasMsg := msg.Extras.NasMsg
	rand := nasMsg.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], snName)

	// TODO: Parse Auth Request IEs and update the RealUE Context

	// Now generate NAS Authentication Response
	ue.Log.Traceln("Generating Authentication Reponse Message")
	nasPdu := nasTestpacket.GetAuthenticationResponse(resStat, "")

	SendToSimUe(ue, common.AUTH_RESPONSE_EVENT, nasPdu, nil)
	ue.Log.Traceln("Sent Authentication Reponse Message to SimUe")
	return nil
}

func HandleSecModCompleteEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Security Mode Complete Event")

	//TODO: Process corresponding Security Mode Command first

	mobileId5GS := nasType.MobileIdentity5GS{
		Len:    uint16(len(ue.Suci)), // suci
		Buffer: ue.Suci,
	}
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileId5GS, nil,
		ue.GetUESecurityCapability(), ue.Get5GMMCapability(), nil, nil)

	ue.Log.Traceln("Generating Security Mode Complete Message")
	nasPdu := nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)

	nasPdu, err = test.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext,
		true, true)
	if err != nil {
		ue.Log.Errorln("Failed to encrypt Security Mode Complete Message", err)
		return err
	}

	SendToSimUe(ue, common.SEC_MOD_COMPLETE_EVENT, nasPdu, nil)
	ue.Log.Traceln("Sent Security Mode Complete Message to SimUe")
	return nil
}

func HandleRegCompleteEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Registration Complete Event")

	//TODO: Process corresponding Registration Accept first

	ue.Log.Traceln("Generating Registration Complete Message")
	nasPdu := nasTestpacket.GetRegistrationComplete(nil)
	nasPdu, err = test.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		ue.Log.Errorln("Failed to encrypt Registration Complete Message", err)
		return
	}

	SendToSimUe(ue, common.REG_COMPLETE_EVENT, nasPdu, nil)
	ue.Log.Traceln("Sent Registration Complete Message to SimUe")
	return nil
}

func HandlePduSessEstRequestEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling PDU Session Establishment Request Event")

	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "010203",
	}
	nasPdu := nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10,
		nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)

	nasPdu, err = test.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		fmt.Println("Failed to encrypt PDU Session Establishment Request Message", err)
		return
	}

	SendToSimUe(ue, common.PDU_SESS_EST_REQUEST_EVENT, nasPdu, nil)
	ue.Log.Traceln("Sent Registration Complete Message to SimUe")
	return nil
}

func HandlePduSessEstAcceptEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling PDU Session Establishment Accept Event")
	//create new pdu session var and parse msg to pdu session var
	return nil
}

func HandleDlInfoTransferEvent(ue *context.RealUe, msg *common.UuMessage) (err error) {
	ue.Log.Traceln("Handling Downlink Nas Transport Event")
	pdu := msg.NasPdu
	nasMsg, err := test.NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
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

	event := common.EventType(msgType)

	// Simply notify SimUe about the received nas message. Later SimUe will
	// asynchrously send next event to RealUE informing about what to do with
	// the received NAS message
	SendToSimUe(ue, event, nil, nasMsg)
	ue.Log.Infoln("Notified SimUe for message type:", msgType)
	return nil
}
