// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package gutiregistration

import (
	"fmt"
	"gnbsim/util/test" // AJAY - Change required
	"time"

	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
)

// Registration -> DeRegistration(UE Originating)
func Gutiregistration_test(ranIpAddr, amfIpAddr string) {
	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	// RAN connect to AMF
	amfConn, err := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9487)
	if err != nil {
		fmt.Println("Failed to connect to AMF ", amfIpAddr)
	} else {
		fmt.Println("Success - connected to AMF ", amfIpAddr)
	}
	// send NGSetupRequest Msg
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "free5gc")
	_, err = amfConn.Write(sendMsg)

	// receive NGSetupResponse Msg
	n, err = amfConn.Read(recvMsg)
	_, err = ngap.Decoder(recvMsg[:n])

	// New UE
	ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC, "")

	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	SUCI5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}
	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, SUCI5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
	_, err = amfConn.Write(sendMsg)

	// receive NAS Authentication Request Msg
	n, err = amfConn.Read(recvMsg)
	ngapMsg, err := ngap.Decoder(recvMsg[:n])

	// Calculate for RES*
	nasPdu := test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	// receive NAS Security Mode Command Msg
	n, err = amfConn.Read(recvMsg)
	_, err = ngap.Decoder(recvMsg[:n])

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		SUCI5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)

	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	// receive ngap Initial Context Setup Request Msg
	n, err = amfConn.Read(recvMsg)
	_, err = ngap.Decoder(recvMsg[:n])

	// send ngap Initial Context Setup Response Msg
	sendMsg, err = test.GetInitialContextSetupResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
	_, err = amfConn.Write(sendMsg)

	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	time.Sleep(500 * time.Millisecond)

	// send NAS Deregistration Request (UE Originating)
	GUTI5GS := nasType.MobileIdentity5GS{
		Len:    11, // 5g-guti
		Buffer: []uint8{0x02, 0x02, 0xf8, 0x39, 0xca, 0xfe, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	pdu = nasTestpacket.GetDeregistrationRequest(nasMessage.AccessType3GPP, 0, 0x04, GUTI5GS)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	time.Sleep(500 * time.Millisecond)

	// receive NAS Deregistration Accept
	n, err = amfConn.Read(recvMsg)
	ngapMsg, err = ngap.Decoder(recvMsg[:n])
	if ngapType.NGAPPDUPresentInitiatingMessage != ngapMsg.Present {
		return
	}
	if ngapType.ProcedureCodeDownlinkNASTransport != ngapMsg.InitiatingMessage.ProcedureCode.Value {
		return
	}
	if ngapType.InitiatingMessagePresentDownlinkNASTransport != ngapMsg.InitiatingMessage.Value.Present {
		return
	}
	nasPdu = test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	if nasPdu == nil {
		return
	}
	if nasPdu.GmmMessage == nil {
		return
	}

	if nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration != nasPdu.GmmMessage.GmmHeader.GetMessageType() {
		return
	}

	// receive ngap UE Context Release Command
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// send ngap UE Context Release Complete
	sendMsg, err = test.GetUEContextReleaseComplete(ue.AmfUeNgapId, ue.RanUeNgapId, nil)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	time.Sleep(200 * time.Millisecond)

	// ========================= Second Registration - Register with GUTI =========================

	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	// innerRegistrationRequest will be encapsulated in the registrationRequest
	innerRegistrationRequest := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		GUTI5GS, nil, ue.GetUESecurityCapability(), ue.Get5GMMCapability(), nil, nil)
	//registrationRequest = nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
	//	GUTI5GS, nil, ueSecurityCapability, nil, innerRegistrationRequest, nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, innerRegistrationRequest, nas.SecurityHeaderTypeIntegrityProtected, true, false)
	if err != nil {
		return
	}
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, pdu, "")
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive NAS Identity Request
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	ngapMsg, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}
	if ngapType.NGAPPDUPresentInitiatingMessage != ngapMsg.Present {
		return
	}
	if ngapType.ProcedureCodeDownlinkNASTransport != ngapMsg.InitiatingMessage.ProcedureCode.Value {
		return
	}
	if ngapType.InitiatingMessagePresentDownlinkNASTransport != ngapMsg.InitiatingMessage.Value.Present {
		return
	}
	nasPdu = test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	if nasPdu == nil {
		return
	}
	if nasPdu.GmmMessage == nil {
		return
	}

	if nas.MsgTypeRegistrationAccept != nasPdu.GmmMessage.GmmHeader.GetMessageType() {
		return
	}
	/*
	   	// send NAS Identity Response
	   	mobileIdentity := nasType.MobileIdentity{
	   		Len:    SUCI5GS.Len,
	   		Buffer: SUCI5GS.Buffer,
	   	}
	   	pdu = nasTestpacket.GetIdentityResponse(mobileIdentity)
	   	if err != nil {
	           return
	       }
	   	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	   	if err != nil {
	           return
	       }
	   	_, err = amfConn.Write(sendMsg)
	   	if err != nil {
	           return
	       }

	   	// receive NAS Authentication Request Msg
	   	n, err = amfConn.Read(recvMsg)
	   	if err != nil {
	           return
	       }
	   	ngapMsg, err = ngap.Decoder(recvMsg[:n])
	       if err != nil {
	           return
	       }
	   	if ngapType.NGAPPDUPresentInitiatingMessage != ngapMsg.Present {
	           return
	       }
	   	if ngapType.ProcedureCodeDownlinkNASTransport != ngapMsg.InitiatingMessage.ProcedureCode.Value {
	           return
	       }
	   	if ngapType.InitiatingMessagePresentDownlinkNASTransport != ngapMsg.InitiatingMessage.Value.Present {
	           return
	       }
	   	nasPdu = test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	   	if nasPdu == nil {
	           return
	       }
	   	if nasPdu.GmmMessage == nil {
	           return
	       }
	   	if nas.MsgTypeAuthenticationRequest != nasPdu.GmmMessage.GmmHeader.GetMessageType() {
	           return
	       }

	   	// Calculate for RES*
	   	rand = nasPdu.AuthenticationRequest.GetRANDValue()
	   	sqn, _ := strconv.ParseUint(ue.AuthenticationSubs.SequenceNumber, 16, 48)
	   	sqn++
	   	ue.AuthenticationSubs.SequenceNumber = strconv.FormatUint(sqn, 16)
	   	resStat = ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	   	// send NAS Authentication Response
	   	pdu = nasTestpacket.GetAuthenticationResponse(resStat, "")
	   	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	       if err != nil {
	           return
	       }
	   	_, err = amfConn.Write(sendMsg)
	       if err != nil {
	           return
	       }

	   	// receive NAS Security Mode Command Msg
	   	n, err = amfConn.Read(recvMsg)
	       if err != nil {
	           return
	       }
	   	ngapMsg, err = ngap.Decoder(recvMsg[:n])
	       if err != nil {
	           return
	       }
	   	if ngapType.NGAPPDUPresentInitiatingMessage != ngapMsg.Present {
	           return;
	       }
	   	if ngapType.ProcedureCodeDownlinkNASTransport != ngapMsg.InitiatingMessage.ProcedureCode.Value {
	           return;
	       }
	   	if ngapType.InitiatingMessagePresentDownlinkNASTransport != ngapMsg.InitiatingMessage.Value.Present {
	           return;
	       }
	   	nasPdu = test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	       if nasPdu == nil {
	           return
	       }
	       if nasPdu.GmmMessage == nil {
	           return
	       }
	   	if nas.MsgTypeSecurityModeCommand != nasPdu.GmmMessage.GmmHeader.GetMessageType() {
	           return
	       }

	   	// send NAS Security Mode Complete Msg
	   	pdu = nasTestpacket.GetSecurityModeComplete(nil)
	   	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	       if err != nil {
	           return
	       }
	   	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	       if err != nil {
	           return
	       }
	   	_, err = amfConn.Write(sendMsg)
	       if err != nil {
	           return
	       }

	   	// receive ngap Initial Context Setup Request Msg
	   	n, err = amfConn.Read(recvMsg)
	       if err != nil {
	           return
	       }
	   	_, err = ngap.Decoder(recvMsg[:n])
	       if err != nil {
	           return
	       }
	*/
	// send ngap Initial Context Setup Response Msg
	sendMsg, err = test.GetInitialContextSetupResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}
	time.Sleep(1000 * time.Millisecond)

	// close Connection
	amfConn.Close()
}
