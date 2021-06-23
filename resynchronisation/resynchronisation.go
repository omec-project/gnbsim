package resynchronisation

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
    "github.com/free5gc/milenage"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"time"
	"bytes"
	"encoding/binary"
)

func Resychronisation_test(ranIpAddr, upfIpAddr,  amfIpAddr string) {
	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	// RAN connect to AMF
	amfConn, err := test.ConnectToAmf(amfIpAddr, ranIpAddr, 38412, 9487)
	if err != nil {
		fmt.Println("Failed to connect to AMF ", amfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to AMF ", amfIpAddr)
	}

	// RAN connect to UPF
	upfConn, err := test.ConnectToUpf(ranIpAddr, upfIpAddr, 2152, 2152)
	if err != nil {
		fmt.Println("Failed to connect to UPF ", upfIpAddr)
        return
	} else {
		fmt.Println("Success - connected to UPF ", upfIpAddr)
	}
	// send NGSetupRequest Msg
	fmt.Println("Send NGSetupRequest Message")
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "free5gc")
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
	fmt.Println("Received NGSetupResponse Message")
	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode NGSetupResponse message")
		return
	}
	// New UE
	// ue := test.NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA2, security.AlgIntegrity128NIA2)
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
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}

	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, err = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
	fmt.Println("Send Initial UE Registration Request Message")
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to write - UE Registration Request Message")
		return
	}

	// receive NAS Authentication Request Msg
	n, err = amfConn.Read(recvMsg)
	ngapMsg, err := ngap.Decoder(recvMsg[:n])

	nasPdu := test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)

	// gen AK
	K, OPC := make([]byte, 16), make([]byte, 16)
	K, _ = hex.DecodeString(ue.AuthenticationSubs.PermanentKey.PermanentKeyValue)
	OPC, _ = hex.DecodeString(ue.AuthenticationSubs.Opc.OpcValue)
	SQN := make([]byte, 6)
	AK := make([]byte, 6)

	rand := nasPdu.AuthenticationRequest.GetRANDValue()
	milenage.F2345(OPC, K, rand[:], nil, nil, nil, AK, nil)
	autn := nasPdu.AuthenticationRequest.GetAUTN()
	SQNxorAK := autn[:6]
	for i := 0; i < 6; i++ {
		SQN[i] = AK[i] ^ SQNxorAK[i]
	}
	const SqnMAx int64 = 0x7FFFFFFFFFF
	const SqnMs int64 = 0
	const IND int64 = 32
	var newSqnMsString string
	SQNBuffer := make([]byte, 8)
	copy(SQNBuffer[2:], SQN)
	r := bytes.NewReader(SQNBuffer)
	var retrieveSqn int64
	if err := binary.Read(r, binary.BigEndian, &retrieveSqn); err != nil {
		fmt.Println("err", err)
		return
	}

	delita := retrieveSqn - SqnMs
	if delita < 0x7FFFFFFFFFF {
		newSqnMsString = "000000000000"
	}

	newSqnMs, _ := hex.DecodeString(newSqnMsString)
	MAC_A, MAC_S := make([]byte, 8), make([]byte, 8)
	CK, IK := make([]byte, 16), make([]byte, 16)
	RES := make([]byte, 8)
	AK, AKstar := make([]byte, 6), make([]byte, 6)
	AMF, _ := hex.DecodeString("0000")
	milenage.F1(OPC, K, rand[:], newSqnMs, AMF, MAC_A, MAC_S)
	milenage.F2345(OPC, K, rand[:], RES, CK, IK, AK, AKstar)

	SQNmsxorAK := make([]byte, 6)
	for i := 0; i < len(SQN); i++ {
		SQNxorAK[i] = SQN[i] ^ AK[i]
	}
	ColSQNmsxorAK := make([]byte, 6)
	for i := 0; i < len(SQN); i++ {
		ColSQNmsxorAK[i] = SQNmsxorAK[i] ^ AKstar[i]
	}
	AUTS := append(ColSQNmsxorAK, MAC_S...)
	// compute SQN by AUTN, K, AK
	// suppose
	// send NAS Authentication Rejcet
	// failureParam := []uint8{0x68, 0x58, 0x15, 0x86, 0x1f, 0xec, 0x0f, 0xa9, 0x48, 0xe8, 0xb2, 0x3a, 0x08, 0x62}
	failureParam := AUTS
	pdu := nasTestpacket.GetAuthenticationFailure(nasMessage.Cause5GMMSynchFailure, failureParam)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	// receive NAS Authentication Request Msg
	n, err = amfConn.Read(recvMsg)
	ngapMsg, err = ngap.Decoder(recvMsg[:n])

	// Calculate for RES*
	nasPdu = test.GetNasPdu(ue, ngapMsg.InitiatingMessage.Value.DownlinkNASTransport)
	rand = nasPdu.AuthenticationRequest.GetRANDValue()

	milenage.F2345(OPC, K, rand[:], nil, nil, nil, AK, nil)
	autn = nasPdu.AuthenticationRequest.GetAUTN()
	SQNxorAK = autn[:6]

	for i := 0; i < 6; i++ {
		SQN[i] = AK[i] ^ SQNxorAK[i]
	}
	fmt.Printf("retrieve SQN %x\n", SQN)
	ue.AuthenticationSubs.SequenceNumber = hex.EncodeToString(SQN)
	resStar := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// send NAS Authentication Response
	pdu = nasTestpacket.GetAuthenticationResponse(resStar, "")
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	// receive NAS Security Mode Command Msg
	n, err = amfConn.Read(recvMsg)
	_, err = ngap.Decoder(recvMsg[:n])

	// send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
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

	time.Sleep(100 * time.Millisecond)
	// send GetPduSessionEstablishmentRequest Msg

	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "010203",
	}
	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
	pdu, err = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, err = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
	_, err = amfConn.Write(sendMsg)

	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
	n, err = amfConn.Read(recvMsg)
	ngapPdu, err := ngap.Decoder(recvMsg[:n])

    nasPdu = test.GetNasPduSetupRequest(ue, ngapPdu.InitiatingMessage.Value.PDUSessionResourceSetupRequest)
    fmt.Println("Assigne address to UE address ", nasPdu.GmmMessage.DLNASTransport.Ipaddr)
    ueIpaddr := nasPdu.GmmMessage.DLNASTransport.Ipaddr


	// send 14. NGAP-PDU Session Resource Setup Response
    var pduSessionId int64
    pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId, ue.AmfUeNgapId, ue.RanUeNgapId, ranIpAddr)
	_, err = amfConn.Write(sendMsg)

	// wait 1s
	time.Sleep(1 * time.Second)

	// Send the dummy packet
	// ping IP(tunnel IP) from 60.60.0.2(127.0.0.1) to 60.60.0.20(127.0.0.8)
	gtpHdr, err := hex.DecodeString("32ff00340000000100000000")
	icmpData, err := hex.DecodeString("8c870d0000000000101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")

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
	tt := append(gtpHdr, v4HdrBuf...)

	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 12394, Seq: 1,
			Data: icmpData,
		},
	}
	b, err := m.Marshal(nil)
	b[2] = 0xaf
	b[3] = 0x88
	_, err = upfConn.Write(append(tt, b...))

	time.Sleep(1 * time.Second)

	// delete test data
	test.DelAuthSubscriptionToMongoDB(ue.Supi)
	test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	test.DelSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)

	// close Connection
	amfConn.Close()
}
