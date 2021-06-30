// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package n2handover

import (
	"encoding/hex"
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
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"time"
	"github.com/mohae/deepcopy"
)

// Registration -> PDU Session Establishment -> Source RAN Send Handover Required -> N2 Handover (Preparation Phase -> Execution Phase)
func N2Handover_test(ranIpAddr, upfIpAddr, amfIpAddr string) {
	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	// RAN1 connect to AMF
	amfConn1, err := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9487)
	if err != nil {
		fmt.Println("Failed to connect to AMF ", amfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to AMF ", amfIpAddr)
	}

	// RAN1 connect to UPF
	upfConn, err := test.ConnectToUpf(ranIpAddr, upfIpAddr, 2152, 2152)
	if err != nil {
		fmt.Println("Failed to connect to UPF ", upfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to UPF ", upfIpAddr)
	}
	// RAN1 send NGSetupRequest Msg
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x01"), 24, "free5gc")
	if err != nil {
		fmt.Println("failed to create setupRequest message")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
		fmt.Println("failed to write setupRequest message")
		return
	}

	// RAN1 receive NGSetupResponse Msg
	fmt.Println("Wait to receive NGSetupResponse Message")
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
		fmt.Println("failed to read NGSetupResponse message")
		return
	}
	fmt.Println("Received NGSetupResponse Message")

	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode NGSetupResponse message")
		return
	}

	time.Sleep(10 * time.Millisecond)

	// RAN2 connect to AMF
	amfConn2, err := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9488)
	if err != nil {
		fmt.Println("Failed to connect to AMF ", amfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to AMF ", amfIpAddr)
	}
	// RAN2 connect to UPF
	upfConn2, err := test.ConnectToUpf(ranIpAddr, upfIpAddr, 2152, 2152)
	if err != nil {
		fmt.Println("Failed to connect to UPF ", upfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to UPF ", upfIpAddr)
	}

	// RAN2 send Second NGSetupRequest Msg
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "nctu")
	if err != nil {
        fmt.Println("GetNGSetupRequest failed")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("GetNGSetupRequest write failed")
		return
	}

	// RAN2 receive Second NGSetupResponse Msg
	n, err = amfConn2.Read(recvMsg)
	if err != nil {
        fmt.Println("NGSetupResponse Msg")
		return
	}
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
        fmt.Println("GetInitialUEMessage failed")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("GetInitialUEMessage write failed")
		return
	}

	// receive NAS Authentication Request Msg
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
        fmt.Println("failed to NAS Authentication Request Msg ")
		return
	}
	ngapMsg, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Authentication message")
		return
	}

	// Calculate for RES*
	nasPdu := test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	if nasPdu == nil {
        fmt.Println("GetNasPdu failed")
        return
    }
	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
        fmt.Println("GetUplinkNASTransport failed")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("GetUplinkNASTransport write failed")
		return
	}

	// receive NAS Security Mode Command Msg
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
        fmt.Println("failed to Read Security Mode Command Msg ")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Security Mode Command message")
		return
	}

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	if err != nil {
        fmt.Println("failure EncodeNasPduWithSecurity ")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
        fmt.Println("failure GetUplinkNASTransport")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure GetUplinkNASTransport write 3")
		return
	}

	// receive ngap Initial Context Setup Request Msg
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to Read Initial Context Setup Request message")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Initial Context Setup Request message")
		return
	}

	// send ngap Initial Context Setup Response Msg
	sendMsg, err = test.GetInitialContextSetupResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
	if err != nil {
        fmt.Println("failure 4")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 5")
		return
	}

	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
        fmt.Println("failure 6")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
        fmt.Println("failure 7")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 8")
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
        fmt.Println("failure 9")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	if err != nil {
        fmt.Println("failure 10")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 11")
		return
	}

	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read PDU session Resource setup req message")
		return
	}
	ngapPdu, err := ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode PDU session Resource setup req message")
		return
	}
    nasPdu = test.GetNasPduSetupRequest(ue, ngapPdu.InitiatingMessage.Value.PDUSessionResourceSetupRequest)
    fmt.Println("Assigne address to UE address ", nasPdu.GmmMessage.DLNASTransport.Ipaddr)
    ueIpaddr := nasPdu.GmmMessage.DLNASTransport.Ipaddr

	// send 14. NGAP-PDU Session Resource Setup Response
    var pduSessionId int64
    pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId, ue.AmfUeNgapId, ue.RanUeNgapId, ranIpAddr)
	if err != nil {
        fmt.Println("failure 12")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 13")
		return
	}

	time.Sleep(1 * time.Second)

	// Send the dummy packet to test if UE is connected to RAN1
	// ping IP(tunnel IP) from 60.60.0.1(127.0.0.1) to 60.60.0.100(127.0.0.8)
	gtpHdr, err := hex.DecodeString("32ff00340000000100000000")
	if err != nil {
        fmt.Println("failure 14")
		return
	}
	icmpData, err := hex.DecodeString("8c870d0000000000101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")
	if err != nil {
        fmt.Println("failure 15")
		return
	}

	ipv4hdr := ipv4.Header{
		Version:  4,
		Len:      20,
		Protocol: 1,
		Flags:    0,
		TotalLen: 48,
		TTL:      64,
		Src:      net.ParseIP(ueIpaddr).To4(),    // ue IP address
		Dst:      net.ParseIP("192.168.250.1").To4(), // upstream router interface connected to Gi
		ID:       1,
	}
	checksum := test.CalculateIpv4HeaderChecksum(&ipv4hdr)
	ipv4hdr.Checksum = int(checksum)

	v4HdrBuf, err := ipv4hdr.Marshal()
	if err != nil {
        fmt.Println("failure 16")
		return
	}
	tt := append(gtpHdr, v4HdrBuf...)
	if err != nil {
        fmt.Println("failure 17")
		return
	}

	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 12394, Seq: 1,
			Data: icmpData,
		},
	}
	b, err := m.Marshal(nil)
	if err != nil {
        fmt.Println("failure 18")
		return
	}
	b[2] = 0xaf
	b[3] = 0x88
	_, err = upfConn.Write(append(tt, b...))
	if err != nil {
        fmt.Println("failure 19")
		return
	}

	time.Sleep(1 * time.Second)

	// ============================================

	// Source RAN send ngap Handover Required Msg
	sendMsg, err = test.GetHandoverRequired(ue.AmfUeNgapId, ue.RanUeNgapId, []byte{0x00, 0x01, 0x02}, []byte{0x01, 0x20})
	if err != nil {
        fmt.Println("failure 20")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 21")
		return
	}

	// Target RAN receive ngap Handover Request
	n, err = amfConn2.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Handover Request message")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Handover Request message")
		return
	}

	// Target RAN create New UE
	targetUe := deepcopy.Copy(ue).(*test.RanUeContext)
	targetUe.AmfUeNgapId = 2
	targetUe.ULCount.Set(ue.ULCount.Overflow(), ue.ULCount.SQN())
	targetUe.DLCount.Set(ue.DLCount.Overflow(), ue.DLCount.SQN())

	// Target RAN send ngap Handover Request Acknowledge Msg
	sendMsg, err = test.GetHandoverRequestAcknowledge(targetUe.AmfUeNgapId, targetUe.RanUeNgapId)
	if err != nil {
        fmt.Println("failure 22")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 23")
		return
	}

	// End of Preparation phase
	time.Sleep(10 * time.Millisecond)

	// Beginning of Execution

	// Source RAN receive ngap Handover Command
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Handover Command message")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Handover Command message")
		return
	}

	// Target RAN send ngap Handover Notify
	sendMsg, err = test.GetHandoverNotify(targetUe.AmfUeNgapId, targetUe.RanUeNgapId)
	if err != nil {
        fmt.Println("failure 24")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 25")
		return
	}

	// Source RAN receive ngap UE Context Release Command
	n, err = amfConn1.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Context Release Command message")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Context Release Command message")
		return
	}

	// Source RAN send ngap UE Context Release Complete
	pduSessionIDList := []int64{10}
	sendMsg, err = test.GetUEContextReleaseComplete(ue.AmfUeNgapId, ue.RanUeNgapId, pduSessionIDList)
	if err != nil {
        fmt.Println("failure 26")
		return
	}
	_, err = amfConn1.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 27")
		return
	}

	// UE send NAS Registration Request(Mobility Registration Update) To Target AMF (2 AMF scenario not supportted yet)
	mobileIdentity5GS = nasType.MobileIdentity5GS{
		Len:    11, // 5g-guti
		Buffer: []uint8{0x02, 0x02, 0xf8, 0x39, 0xca, 0xfe, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	uplinkDataStatus := nasType.NewUplinkDataStatus(nasMessage.RegistrationRequestUplinkDataStatusType)
	uplinkDataStatus.SetLen(2)
	uplinkDataStatus.SetPSI10(1)
	ueSecurityCapability = targetUe.GetUESecurityCapability()
	pdu = nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSMobilityRegistrationUpdating,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, uplinkDataStatus)
	pdu, err = test.EncodeNasPduWithSecurity(targetUe, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
        fmt.Println("failure EncodeNasPduWithSecurity")
		return
	}
	sendMsg, err = test.GetInitialUEMessage(targetUe.RanUeNgapId, pdu, "")
	if err != nil {
        fmt.Println("failure GetInitialUEMessage ")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("failure write to socket 1")
		return
	}

	// Target RAN receive ngap Initial Context Setup Request Msg
	n, err = amfConn2.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read Initial Context Setup Request message")
		return
	}
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode Initial Context Setup Request message")
		return
	}

	// Target RAN send ngap Initial Context Setup Response Msg
	sendMsg, err = test.GetInitialContextSetupResponseForServiceRequest(targetUe.AmfUeNgapId, targetUe.RanUeNgapId, "10.200.200.2")
	if err != nil {
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("failure 1")
		return
	}

	// Target RAN send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = test.EncodeNasPduWithSecurity(targetUe, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
        fmt.Println("failure 1")
		return
	}
	sendMsg, err = test.GetUplinkNASTransport(targetUe.AmfUeNgapId, targetUe.RanUeNgapId, pdu)
	if err != nil {
        fmt.Println("failure GetUplinkNASTransport")
		return
	}
	_, err = amfConn2.Write(sendMsg)
	if err != nil {
        fmt.Println("failure Writing UplinkNASTrasport")
		return
	}

	// wait 1000 ms
	time.Sleep(1000 * time.Millisecond)

	// Send the dummy packet
	// ping IP(tunnel IP) from 60.60.0.2(127.0.0.1) to 60.60.0.20(127.0.0.8)
	_, err = upfConn2.Write(append(tt, b...))
	if err != nil {
        fmt.Println("failure sending dummy GTPU packet ")
		return
	}

	time.Sleep(100 * time.Millisecond)

	// delete test data
	test.DelAuthSubscriptionToMongoDB(ue.Supi)
	test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	test.DelSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)

	// close Connection
	amfConn1.Close()
	amfConn2.Close()
}
