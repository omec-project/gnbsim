// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package  pdusessionrelease

import (
	"fmt"
	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/nas/nasTestpacket"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/ngap"
	"github.com/free5gc/openapi/models"
    "gnbsim/util/test" // AJAY - Change required 
	"time"
)

// Registration -> Pdu Session Establishment -> Pdu Session Release
func PduSessionRelease_test(ranIpAddr,  amfIpAddr string) {
	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	// RAN connect to AMF
	amfConn, err := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9487)

	// send NGSetupRequest Msg
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "free5gc")
	_, err = amfConn.Write(sendMsg)

	// receive NGSetupResponse Msg
    fmt.Println("Read NGSetupResponse Msg")
	n, err = amfConn.Read(recvMsg)
    if err != nil {
        fmt.Println("failed to read message")
        return
    }

	_, err = ngap.Decoder(recvMsg[:n])
    if err != nil {
        fmt.Println("failed to decode message")
        return
    }

	// New UE
	ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC,
		TestGenAuthData.MilenageTestSet19.OP)
	// insert UE data to MongoDB

	servingPlmnId := "20893"
	test.InsertAuthSubscriptionToMongoDB(ue.Supi, ue.AuthenticationSubs)
	//getData := test.GetAuthSubscriptionFromMongoDB(ue.Supi)
	{
		amData := test.GetAccessAndMobilitySubscriptionData()
		test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
		//getData := test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		smfSelData := test.GetSmfSelectionSubscriptionData()
		test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
		//getData := test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		smSelData := test.GetSessionManagementSubscriptionData()
		test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
		//getData := test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		amPolicyData := test.GetAmPolicyData()
		test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
		//getData := test.GetAmPolicyDataFromMongoDB(ue.Supi)
	}
	{
		smPolicyData := test.GetSmPolicyData()
		test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
		//getData := test.GetSmPolicyDataFromMongoDB(ue.Supi)
	}

	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}
	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
	_, err = amfConn.Write(sendMsg)

	// receive NAS Authentication Request Msg
    fmt.Println("Read Authentication Request Msg")
	n, err = amfConn.Read(recvMsg)
    if err != nil {
        fmt.Println("failed to read message")
        return
    }
	ngapMsg, err := ngap.Decoder(recvMsg[:n])
    if err != nil {
        fmt.Println("failed to decode message")
        return
    }

	// Calculate for RES*
	nasPdu := test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// receive NAS Security Mode Command Msg
    fmt.Println("Read NAS Security Mode Command Msg")
	n, err = amfConn.Read(recvMsg)
    if err != nil {
        fmt.Println("failed to read message")
        return
    }

	_, err = ngap.Decoder(recvMsg[:n])
    if err != nil {
        fmt.Println("failed to decode message")
        return
    }

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// receive ngap Initial Context Setup Request Msg
    fmt.Println("Read Initial Context Setup Request Msg")
	n, err = amfConn.Read(recvMsg)
    if err != nil {
        fmt.Println("failed to read message")
        return
    }

	_, err = ngap.Decoder(recvMsg[:n])
    if err != nil {
        fmt.Println("failed to decode message")
        return
    }

	// send ngap Initial Context Setup Response Msg
	sendMsg, err = test.GetInitialContextSetupResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// send PduSessionEstablishmentRequest Msg

	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "010203",
	}
	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
    fmt.Println("Read NAS PDU Session Resource setup Request Msg")
	n, err = amfConn.Read(recvMsg)
    if err != nil {
        fmt.Println("failed to read message")
        return
    }

	_, err = ngap.Decoder(recvMsg[:n])
    if err != nil {
        fmt.Println("failed to decode message")
        return
    }

	// send 14. NGAP-PDU Session Resource Setup Response
    var pduSessionId int64
    pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId, ue.AmfUeNgapId, ue.RanUeNgapId, ranIpAddr)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// Send Pdu Session Establishment Release Request
	pdu = nasTestpacket.GetUlNasTransport_PduSessionReleaseRequest(10)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	time.Sleep(1000 * time.Millisecond)
	// send N2 Resource Release Ack(PDUSession Resource Release Response)
	sendMsg, err = test.GetPDUSessionResourceReleaseResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// wait 10 ms
	time.Sleep(1000 * time.Millisecond)

	//send N1 PDU Session Release Ack PDU session release complete
	pdu = nasTestpacket.GetUlNasTransport_PduSessionReleaseComplete(10, nasMessage.ULNASTransportRequestTypeExistingPduSession, "internet", &sNssai)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)
    if err != nil {
        fmt.Println("failed to write message")
        return
    }

	// wait result
	time.Sleep(1 * time.Second)

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
	if nasPdu != nil {
		fmt.Println("NAS PDU is nil")
		return
	}
	if nasPdu.GmmMessage != nil {
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
	fmt.Println("sent UE Context Release Complete")

	time.Sleep(100 * time.Millisecond)


	// delete test data
	test.DelAuthSubscriptionToMongoDB(ue.Supi)
	test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	test.DelSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)

	// close Connection
	amfConn.Close()
}
