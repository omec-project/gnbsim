// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package servicereq

import (
	"fmt"
	"gnbsim/util/test" // AJAY - Change required
	"time"

	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
)

// Registration -> Pdu Session Establishment -> AN Release due to UE Idle -> UE trigger Service Request Procedure
func Servicereq_test(ranIpAddr, upfIpAddr, amfIpAddr string) {
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
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive NGSetupResponse Msg
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// New UE
	ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC, "")
	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}
	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
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
	ngapMsg, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// Calculate for RES*
	nasPdu := test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	ue.AmfUeNgapId = ngapMsg.InitiatingMessage.Value.DownlinkNASTransport.ProtocolIEs.List[0].Value.AMFUENGAPID.Value
	if nasPdu == nil {
		return
	}
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")
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
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
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

	// send PduSessionEstablishmentRequest Msg

	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "010203",
	}
	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
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

	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// send 14. NGAP-PDU Session Resource Setup Response
	var pduSessionId int64
	pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId, ue.AmfUeNgapId, ue.RanUeNgapId, ranIpAddr)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}
	// send NAS Deregistration Request (UE Originating)
	mobileIdentity5GS = nasType.MobileIdentity5GS{
		Len:    11, // 5g-guti
		Buffer: []uint8{0x02, 0x02, 0xf8, 0x39, 0xca, 0xfe, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	pdu = nasTestpacket.GetDeregistrationRequest(nasMessage.AccessType3GPP, 0, 0x04, mobileIdentity5GS)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		fmt.Println("encode NAS DeRegistration Request Message failed")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
		fmt.Println("encode Uplink - NAS DeRegistration Request Message failed")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("write NAS DeRegistration Request Message failed")
		return
	}
	fmt.Println("Send NAS DeRegistration Request Message")

	time.Sleep(500 * time.Millisecond)

	// receive Deregistration Accept
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Read NAS DeRegistration Accept Message failed ")
		return
	}
	ngapPdu, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Decode NAS DeRegistration Accept Message failed ")
		return
	}
	if ngapPdu.Present != ngapType.NGAPPDUPresentInitiatingMessage ||
		ngapPdu.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeDownlinkNASTransport {
		fmt.Println("No DownlinkNASTransport received.")
		return
	}
	fmt.Println("NAS DeRegistration Accept Message received ")

	nasPdu = test.GetNasPdu(ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)
	if nasPdu == nil {
		fmt.Println("NAS PDU is nil")
		return
	}
	if nasPdu.GmmMessage == nil {
		fmt.Println("GMM Message is nil")
		return
	}
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeDeregistrationAcceptUEOriginatingDeregistration {
		fmt.Println("Received wrong GMM message")
		return
	}

	// receive ngap UE Context Release Command
	fmt.Println("waiting for - ngap UE Context Release COmmand ")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Receive UE Context Release Command failed")
		return
	}
	ngapPdu, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Decoder UE Context Release Command failed")
		return
	}
	if ngapPdu.Present == ngapType.NGAPPDUPresentInitiatingMessage ||
		ngapPdu.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeUEContextRelease {
		fmt.Println("No UEContextReleaseCommand received.")
		return
	}
	fmt.Println("ngap UE Context Release Command received ")

	// send ngap UE Context Release Complete
	sendMsg, err = test.GetUEContextReleaseComplete(ue.AmfUeNgapId, ue.RanUeNgapId, nil)
	if err != nil {
		fmt.Println("create UE Context Release Complete failed")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("write UE Context Release Complete failed")
		return
	}
	fmt.Println("sent UE Context Release Complete received")

	time.Sleep(100 * time.Millisecond)

	// send ngap UE Context Release Request
	/*pduSessionIDList := []int64{10}
	sendMsg, err = test.GetUEContextReleaseRequest(ue.AmfUeNgapId, ue.RanUeNgapId, pduSessionIDList)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive UE Context Release Command
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

	// UE is CM-IDLE now

	time.Sleep(1 * time.Second)
	*/
	// send NAS Service Request
	pdu = nasTestpacket.GetServiceRequest(nasMessage.ServiceTypeData)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		return
	}
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, pdu, "fe0000000001")
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive Initial Context Setup Request
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// Send Initial Context Setup Response
	sendMsg, err = test.GetInitialContextSetupResponseForServiceRequest(ue.AmfUeNgapId, ue.RanUeNgapId, ranIpAddr)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	time.Sleep(1 * time.Second)

	// close Connection
	amfConn.Close()
}

// Registration -> Pdu Session Establishment -> AN Release due to UE Idle -> UE trigger Service Request Procedure
func Servicereq_macfail_test(ranIpAddr, upfIpAddr, amfIpAddr string) {
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
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive NGSetupResponse Msg
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// New UE
	ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC, "")

	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}
	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
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
	ngapMsg, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// Calculate for RES*
	nasPdu := test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	if nasPdu == nil {
		return
	}
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")
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
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
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

	// send PduSessionEstablishmentRequest Msg

	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "010203",
	}
	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
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

	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	// send 14. NGAP-PDU Session Resource Setup Response
	var pduSessionId int64
	pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId, ue.AmfUeNgapId, ue.RanUeNgapId, ranIpAddr)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// send ngap UE Context Release Request
	pduSessionIDList := []int64{10}
	sendMsg, err = test.GetUEContextReleaseRequest(ue.AmfUeNgapId, ue.RanUeNgapId, pduSessionIDList)
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive UE Context Release Command
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

	// UE is CM-IDLE now

	time.Sleep(1 * time.Second)

	// send NAS Service Request
	pdu = nasTestpacket.GetServiceRequest(nasMessage.ServiceTypeData)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtected, true, false)
	pdu[2] = 0x00
	pdu[3] = 0x01
	pdu[4] = 0x02
	pdu[5] = 0x03
	if err != nil {
		return
	}
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, pdu, "fe0000000001")
	if err != nil {
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		return
	}

	// receive Initial Context Setup Request
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		return
	}

	time.Sleep(1 * time.Second)

	// close Connection
	amfConn.Close()
}
