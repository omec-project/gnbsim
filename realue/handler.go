// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package realue

import (
	intfc "gnbsim/interfacecommon"
	"gnbsim/realue/context"
	"gnbsim/util/test"

	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasTestpacket"
	"github.com/free5gc/nas/nasType"
	"github.com/omec-project/nas"
)

//TODO Remove the hardcoding
var snName string = "5G:mnc093.mcc208.3gppnetwork.org"

func HandleRegReqEvent(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	ue.Log.Debugln("Handling Registration Request Event")

	ueSecurityCapability := ue.GetUESecurityCapability()
	mobileId5GS := nasType.MobileIdentity5GS{
		Len:    uint16(len(ue.Suci)), // suci
		Buffer: ue.Suci,
	}

	ue.Log.Debugln("Generating Registration Request Message")
	nasPdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileId5GS, nil, ueSecurityCapability, nil, nil, nil)

	SendToSimUe(ue, intfc.UE_REG_REQUEST, nasPdu, nil)
	ue.Log.Debugln("Sent Registration Request Message to SimUe")
	return nil
}

func HandleAuthResponseEvent(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	ue.Log.Debugln("Handling Authentication Response Event")

	// First process the corresponding Auth Request
	ue.Log.Debugln("Processing corresponding Authentication Request Message")
	nasMsg := msg.Extras.NasMsg
	rand := nasMsg.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], snName)

	// TODO: Parse Auth Request IEs and update the RealUE Context

	// Now generate NAS Authentication Response
	ue.Log.Debugln("Generating Authentication Reponse Message")
	nasPdu := nasTestpacket.GetAuthenticationResponse(resStat, "")

	SendToSimUe(ue, intfc.UE_AUTH_RESPONSE, nasPdu, nil)
	ue.Log.Debugln("Sent Authentication Reponse Message to SimUe")
	return nil
}

func HandleSecModCompleteEvent(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	ue.Log.Debugln("Handling Security Mode Complete Event")

	//TODO: Process corresponding Security Mode Command first

	mobileId5GS := nasType.MobileIdentity5GS{
		Len:    uint16(len(ue.Suci)), // suci
		Buffer: ue.Suci,
	}
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileId5GS, nil,
		ue.GetUESecurityCapability(), ue.Get5GMMCapability(), nil, nil)

	ue.Log.Debugln("Generating Security Mode Complete Message")
	nasPdu := nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)

	nasPdu, err = test.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext,
		true, true)
	if err != nil {
		ue.Log.Errorln("Failed to encrypt Security Mode Complete Message", err)
		return err
	}

	SendToSimUe(ue, intfc.UE_SEC_MOD_COMPLETE, nasPdu, nil)
	ue.Log.Debugln("Sent Security Mode Complete Message to SimUe")
	return nil
}

func HandleRegCompleteEvent(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	ue.Log.Debugln("Handling Registration Complete Event")

	//TODO: Process corresponding Registration Accept first

	ue.Log.Debugln("Generating Registration Complete Message")
	nasPdu := nasTestpacket.GetRegistrationComplete(nil)
	nasPdu, err = test.EncodeNasPduWithSecurity(ue, nasPdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		ue.Log.Errorln("Failed to encrypt Registration Complete Message", err)
		return
	}

	SendToSimUe(ue, intfc.UE_REG_COMPLETE, nasPdu, nil)
	ue.Log.Debugln("Sent Registration Complete Message to SimUe")
	return nil
}

func HandleDownlinkNasTransportEvent(ue *context.RealUe, msg *intfc.UuMessage) (err error) {
	ue.Log.Debugln("Handling Downlink Nas Transport Event")
	pdu := msg.NasPdu
	nasMsg, err := test.NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
	if err != nil {
		ue.Log.Errorln("Failed to decode dowlink NAS Message due to", err)
		return err
	}
	msgType := nasMsg.GmmHeader.GetMessageType()
	ue.Log.Infoln("Received Message Type:", msgType)

	event := intfc.EventType(msgType)

	// Simply notify SimUe about the received nas message. Later SimUe will
	// asynchrously send next event to RealUE informing about what to do with
	// the received NAS message
	SendToSimUe(ue, event, nil, nasMsg)
	ue.Log.Infoln("Notified SimUe for message type:", msgType)
	return nil
}
