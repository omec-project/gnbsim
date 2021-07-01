// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package deregister

import (
	"fmt"
	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
    "gnbsim/util/test" // AJAY - Change required 
	"strconv"
	"time"
)

func Deregister_test(ranIpAddr, amfIpAddr string) {
	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	amfConn, err := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9487)
	if err != nil {
		fmt.Println("Failed to connect to AMF ", amfIpAddr)
	} else {
		fmt.Println("Success - connected to AMF ", amfIpAddr)
	}

	ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
		TestGenAuthData.MilenageTestSet19.OPC,
		TestGenAuthData.MilenageTestSet19.OP)

	fmt.Println("Insert Auth Subscription data to MongoDB")
	test.InsertAuthSubscriptionToMongoDB(ue.Supi, ue.AuthenticationSubs)
	//getData := test.GetAuthSubscriptionFromMongoDB(ue.Supi)

	servingPlmnId := "20893"

	{
		fmt.Println("Insert Access & Mobility Subscription data to MongoDB")
		amData := test.GetAccessAndMobilitySubscriptionData()
		test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
		//getData := test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		fmt.Println("Insert SMF Selection Subscription data to MongoDB")
		smfSelData := test.GetSmfSelectionSubscriptionData()
		test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
		//getData := test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		fmt.Println("Insert Session Management Subscription data to MongoDB")
		smSelData := test.GetSessionManagementSubscriptionData()
		test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
		//getData := test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		fmt.Println("Insert Access mobility Policy data to MongoDB")
		amPolicyData := test.GetAmPolicyData()
		test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
		//getData := test.GetAmPolicyDataFromMongoDB(ue.Supi)
	}
	{
		fmt.Println("Insert Session Management Policy data to MongoDB")
		smPolicyData := test.GetSmPolicyData()
		test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
		//getData := test.GetSmPolicyDataFromMongoDB(ue.Supi)
	}

	// send NGSetupRequest Msg
	fmt.Println("Send NGSetupRequest Message")
	sendMsg, _ = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "free5gc")
	amfConn.Write(sendMsg)

	// receive NGSetupResponse Msg
	fmt.Println("Wait to receive NGSetupResponse Message")
	n, _ = amfConn.Read(recvMsg)
	fmt.Println("Received NGSetupResponse Message")
	ngap.Decoder(recvMsg[:n])

	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}

	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, _ = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
	fmt.Println("Send Initial UE Registration Request Message")
	amfConn.Write(sendMsg)

	// receive NAS Authentication Request Msg
	fmt.Println("Wait to receive NAS Authentication Request Message")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read - NAS Authentication Request Message")
		return
	}
	fmt.Println("Received NAS Authentication Request Message")

	ngapPdu, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode NAS Authentication Request msg")
		return
	}

	if ngapPdu.Present != ngapType.NGAPPDUPresentInitiatingMessage {
		fmt.Println("NGAP Initiating Message received.- failed")
		return
	}

	// Calculate for RES*
	nasPdu := test.GetNasPdu(ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)
	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeAuthenticationRequest {
		fmt.Println("Message is not Authentication Request received, but expected Auth Req ")
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

	ngapPdu, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode  Security Mode Command message")
		return
	}

	nasPdu = test.GetNasPdu(ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)

	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeSecurityModeCommand {
		fmt.Println("No Security Mode Command received. Message: " + strconv.Itoa(int(nasPdu.GmmHeader.GetMessageType())))
		return
	}
	fmt.Println("Received Security Mode Command Message")

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

	// receive ngap Initial Context Setup Request Msg
	fmt.Println("Waiting for - Initial Context Setup Request Message")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Initial Context Setup Request Message")
		return
	}

	ngapPdu, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode  Initial Context Setup Request Message ")
		return
	}

	if ngapPdu.Present != ngapType.NGAPPDUPresentInitiatingMessage ||
		ngapPdu.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodeInitialContextSetup {
		fmt.Println("Wrong message received ? or procedure code did not match")
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
	fmt.Println("Send Registration Complete Message")

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
	ngapPdu, err = ngap.Decoder(recvMsg[:n])
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
	fmt.Println("sent UE Context Release Complete failed")

	time.Sleep(100 * time.Millisecond)

	// delete test data
	test.DelAuthSubscriptionToMongoDB(ue.Supi)
	test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	test.DelSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)

	// close Connection
	amfConn.Close()

	return
}
