// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package xnhandover

import (
	"fmt"
	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasTestpacket"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/ngap"
	"github.com/free5gc/openapi/models"
    "gnbsim/util/test" // AJAY - Change required 
	"time"
)

// Registration -> Pdu Session Establishment -> Path Switch(Xn Handover)
func Xnhandover_test(ranUIpAddr, ranIpAddr, upfIpAddr, amfIpAddr string) {
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
	fmt.Println("Send NGSetupRequest Message")
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x01"), 24, "free5gc")
	if err != nil {
		fmt.Println("failed to create setupRequest message")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("failed to write setupRequest message")
		return
	}

	// receive NGSetupResponse Msg
	fmt.Println("Wait to receive NGSetupResponse Message")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read NGSetupResponse message")
		return
	}
	fmt.Println("Received NGSetupResponse Message")
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode NGSetupResponse message")
		return
	}

	time.Sleep(10 * time.Millisecond)

	amfConn2, err1 := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9488)
	if err1 != nil {
		fmt.Println("Failed to connect to AMF ", amfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to AMF ", amfIpAddr)
	}
	// send Second NGSetupRequest Msg
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "nctu")
	if err != nil {
		fmt.Println("failed to create setupRequest message")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
		fmt.Println("failed to write setupRequest message")
		return
	}

	// receive Second NGSetupResponse Msg
	fmt.Println("Wait to receive NGSetupResponse Message")
	n, err = amfConn2.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read NGSetupResponse message")
		return
	}
	fmt.Println("Received NGSetupResponse Message")
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode NGSetupResponse message")
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
	getData := test.GetAuthSubscriptionFromMongoDB(ue.Supi)
	if getData == nil {
		return
	}
	{
		amData := test.GetAccessAndMobilitySubscriptionData()
		test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
		getData := test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
		if getData == nil {
			return
		}
	}
	{
		smfSelData := test.GetSmfSelectionSubscriptionData()
		test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
		getData := test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
		if getData == nil {
			return
		}
	}
	{
		smSelData := test.GetSessionManagementSubscriptionData()
		test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
		getData := test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
		if getData == nil {
			return
		}
	}
	{
		amPolicyData := test.GetAmPolicyData()
		test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
		getData := test.GetAmPolicyDataFromMongoDB(ue.Supi)
		if getData == nil {
			return
		}
	}
	{
		smPolicyData := test.GetSmPolicyData()
		test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
		getData := test.GetSmPolicyDataFromMongoDB(ue.Supi)
		if getData == nil {
			return
		}
	}

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
	    fmt.Println("Failed to get Initial UE Registration Request Message")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
	    fmt.Println("Failed to write Initial UE Registration Request Message")
		return
	}
	fmt.Println("Sent Initial UE Registration Request Message")

	// receive NAS Authentication Request Msg
	fmt.Println("Wait to receive NAS Authentication Request Message")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read - NAS Authentication Request Message")
		return
	}
	ngapMsg, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode - NAS Authentication Request Message")
		return
	}
	fmt.Println("Received NAS Authentication Request Message")

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
		fmt.Println("Failed to create NAS Authentication Response message")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to send NAS Authentication Response message")
		return
	}
	fmt.Println("Sent NAS Authentication Response Message")

	// receive NAS Security Mode Command Msg
	fmt.Println("Waiting for - Security Mode Command Message")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Security Mode Command Message from socket")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode  Security Mode Command message")
		return
	}

	fmt.Println("decode  Security Mode Command message success")
	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	if err != nil {
		fmt.Println("Failed to encode - NAS Security Mode Complete Message")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
		fmt.Println("Failed to create Uplink NAS transport Message")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to write Security Mode Complete Message")
		return
	}

	fmt.Println("sent Security Mode Complete Message")
	fmt.Println("Waiting for - Initial Context Setup Request Message")
	// receive ngap Initial Context Setup Request Msg
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Initial Context Setup Request Message")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode  Initial Context Setup Request Message ")
		return
	}

	// send ngap Initial Context Setup Response Msg
	sendMsg, err = test.GetInitialContextSetupResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
	if err != nil {
		fmt.Println("Failed to get - Initial Context Setup Response Message ")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to write - Initial Context Setup Response Message")
		return
	}

	fmt.Println("Send Initial Context Setup Response Message")
	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		fmt.Println("Failed to encode  NAS PDU - Registration Complete Message ")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
		fmt.Println("Failed to encode  NAS PDU - Registration Complete Message inside Uplink NAS transport")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to send Registration Complete message ")
		return
	}
	fmt.Println("sent Registration Complete message ")

	// send PduSessionEstablishmentRequest Msg

	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "010203",
	}
	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		fmt.Println("Failed to encode NAS PDU Session Establishment Request Message")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
		fmt.Println("Failed to encode NAS PDU Session Establishment Request Message inside Uplink Transport")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to write NAS message - PDU Session Est Req Message")
		return
	}

	fmt.Println("Sent NAS message - PDU Session Est Req Message")
	fmt.Println("waiting for - NGAP-PDU Session Resource Setup Request")
	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read - NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))")
		return
	}
	fmt.Println("Received Message decoding ")
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode - NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))")
		return
	}

	// send 14. NGAP-PDU Session Resource Setup Response
    var pduSessionId int64
    pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId,
                                                           ue.AmfUeNgapId,
                                                           ue.RanUeNgapId, ranUIpAddr)
	if err != nil {
		fmt.Println("Failed to create - NGAP-PDU Session Resource Setup Response")
		return
	}
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to write message NGAP-PDU Session Resource Setup Response ")
		return
	}

	fmt.Println("sent message NGAP-PDU Session Resource Setup Response ")
	time.Sleep(2000 * time.Millisecond)
	// send Path Switch Request (XnHandover)
	sendMsg, err = test.GetPathSwitchRequest(ue.AmfUeNgapId, ue.RanUeNgapId)
	if err != nil {
        fmt.Println("GetPathSwitchRequest failed")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("Failed to write PathSwitchRequest ")
		return
	}

    fmt.Println("Sent PathSwitchRequest ")
	// receive Path Switch Request (XnHandover)
	n, err = amfConn2.Read(recvMsg)
	if err != nil {
	    fmt.Println("Failed to read - Path Switch Request (XnHandover)")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
	    fmt.Println("Failed to decode - Path Switch Request (XnHandover)")
		return
	}

	time.Sleep(10 * time.Millisecond)

	// delete test data
	test.DelAuthSubscriptionToMongoDB(ue.Supi)
	test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	test.DelSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)

	time.Sleep(10000 * time.Millisecond)
	// close Connection
	amfConn.Close()
	amfConn2.Close()
	fmt.Println("Success XnHandover")
}
