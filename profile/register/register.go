// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package register

import (
	"gnbsim/gnodeb"
	"gnbsim/gnodeb/context"
	intfc "gnbsim/interfacecommon"
	"gnbsim/simue"
	simuectx "gnbsim/simue/context"
	"gnbsim/util/test" // AJAY - Change required
	"log"
	"strconv"

	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
)

func Register_test(ranUIpAddr, upfIpAddr string, gnb *context.GNodeB) {

	simUe := simuectx.NewSimUe(gnb)
	simue.Init(simUe)

	gnbUeInboundChan := make(chan *intfc.UuMessage)
	uemsg := intfc.UuMessage{
		UeChan: gnbUeInboundChan,
	}

	gnbUeOutboundChan := gnodeb.RegisterUe(gnb, &uemsg)
	if gnbUeOutboundChan == nil {
		log.Println("Error: GnbUe Channel is nil")
		return
	}

	ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC,
		"")

	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}

	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)

	uemsg = intfc.UuMessage{}
	uemsg.Event = intfc.UE_REG_REQ
	uemsg.Interface = intfc.UU_INTERFACE
	uemsg.NasPdu = registrationRequest
	uemsg.Supi = ue.Supi
	gnbUeOutboundChan <- &uemsg
	incomingMsg := <-gnbUeInboundChan
	if incomingMsg == nil {
		log.Println("Error: Received empty message from GNodeB")
		return
	}
	log.Println("Received NAS Authentication Request Message", incomingMsg)

	// Calculate for RES*
	pdu := incomingMsg.NasPdu
	nasPdu, err := test.NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
	if err != nil {
		log.Println("Error Decoding NAS PDU")
		return
	}
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeAuthenticationRequest {
		log.Println("Message is not Authentication Request received, but expected Auth Req ")
		return
	}
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu = nasTestpacket.GetAuthenticationResponse(resStat, "")

	uemsg = intfc.UuMessage{}
	uemsg.Event = intfc.UE_UPLINK_NAS_TRANSPORT
	uemsg.Interface = intfc.UU_INTERFACE
	uemsg.NasPdu = pdu
	gnbUeOutboundChan <- &uemsg

	log.Println("Sent NAS Authentication Response Message")
	log.Println("Waiting for - Security Mode Command Message")

	incomingMsg = <-gnbUeInboundChan
	if incomingMsg == nil {
		log.Println("Error: Received empty interface message from GNodeB")
		return
	}

	pdu = incomingMsg.NasPdu
	nasPdu, err = test.NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
	if err != nil {
		log.Println("Error Decoding NAS PDU")
		return
	}

	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeSecurityModeCommand {
		log.Println("No Security Mode Command received. Message: " + strconv.Itoa(int(nasPdu.GmmHeader.GetMessageType())))
		return
	}
	log.Println("Received Security Mode Command Message")
	log.Println("Security Mode Command -nasPdu ", nasPdu)

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	if err != nil {
		log.Println("Failed to encode - NAS Security Mode Complete Message")
		return
	}

	uemsg = intfc.UuMessage{}
	uemsg.Event = intfc.UE_UPLINK_NAS_TRANSPORT
	uemsg.Interface = intfc.UU_INTERFACE
	uemsg.NasPdu = pdu
	gnbUeOutboundChan <- &uemsg

	log.Println("sent Security Mode Complete Message")
	log.Println("Waiting for Registration Accept Message")

	incomingMsg = <-gnbUeInboundChan
	if incomingMsg == nil {
		log.Println("Error: Received empty interface message from GNodeB")
		return
	}

	pdu = incomingMsg.NasPdu
	nasPdu, err = test.NASDecode(ue, nas.GetSecurityHeaderType(pdu), pdu)
	if err != nil {
		log.Println("Error Decoding NAS PDU")
		return
	}

	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeRegistrationAccept {
		log.Println("No Registration Accept Received. Message: " + strconv.Itoa(int(nasPdu.GmmHeader.GetMessageType())))
		return
	}
	log.Println("Received Registration Accept Message")

	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		log.Println("Failed to encode  NAS PDU - Registration Complete Message ")
		return
	}

	uemsg = intfc.UuMessage{}
	uemsg.Event = intfc.UE_UPLINK_NAS_TRANSPORT
	uemsg.Interface = intfc.UU_INTERFACE
	uemsg.NasPdu = pdu
	gnbUeOutboundChan <- &uemsg

	log.Println("sent Registration Complete message")
}
