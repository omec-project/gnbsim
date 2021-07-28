// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package register

import (
	"encoding/hex"
	"fmt"
    "gnbsim/util/test" // AJAY - Change required 
	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/omec-project/nas"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"strconv"
	"time"
)

func Register_test(ranUIpAddr, ranIpAddr, upfIpAddr, amfIpAddr string) {

	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	upfConn, err := test.ConnectToUpf(ranUIpAddr, upfIpAddr, 2152, 2152)
	if err != nil {
		fmt.Println("Failed to connect to UPF ", upfIpAddr)
	} else {
		fmt.Println("Success - connected to UPF ", upfIpAddr)
	}

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
		"")

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
	n, _ = amfConn.Read(recvMsg)
	fmt.Println("Received NGSetupResponse Message")

	_, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode NGSetupResponse message")
		return
	}

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
	_, err = amfConn.Write(sendMsg)
	if err != nil {
		fmt.Println("Failed to write - UE Registration Request Message")
		return
	}

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

	dlpdu := ngapPdu.InitiatingMessage.Value.DownlinkNASTransport

	for i := 0; i < len(dlpdu.ProtocolIEs.List); i++ {
		ie := dlpdu.ProtocolIEs.List[i]
		fmt.Println(" IE received ", ie.Id.Value )
        switch(ie.Id.Value) {
            case ngapType.ProtocolIEIDRANUENGAPID:
		        fmt.Println(" NGAP RAN Id received ", ie.Value.RANUENGAPID )
            case ngapType.ProtocolIEIDAMFUENGAPID:
		        fmt.Println(" NGAP AMF Id received ", ie.Value.AMFUENGAPID )
        }
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
	fmt.Println("decode  Security Mode Command message success")

	nasPdu = test.GetNasPdu(ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)

	if nasPdu.GmmHeader.GetMessageType() != nas.MsgTypeSecurityModeCommand {
		fmt.Println("No Security Mode Command received. Message: " + strconv.Itoa(int(nasPdu.GmmHeader.GetMessageType())))
		return
	}
	fmt.Println("Received Security Mode Command Message")
    fmt.Println("Security Mode Command -nasPdu ", nasPdu)

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
	fmt.Println("sent Registration Complete message ")

	time.Sleep(100 * time.Millisecond)
	// send GetPduSessionEstablishmentRequest Msg

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

	// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
	fmt.Println("waiting for - NGAP-PDU Session Resource Setup Request")
	n, err = amfConn.Read(recvMsg)
	if err != nil {
		fmt.Println("Failed to read - NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))")
		return
	}
	fmt.Println("Received Message decoding ")
	ngapPdu, err = ngap.Decoder(recvMsg[:n])
	if err != nil {
		fmt.Println("Failed to decode - NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))")
		return
	}

	if ngapPdu.Present != ngapType.NGAPPDUPresentInitiatingMessage ||
		ngapPdu.InitiatingMessage.ProcedureCode.Value != ngapType.ProcedureCodePDUSessionResourceSetup {
		fmt.Println("Received message type does not match expected message/procedure ", ngapPdu.Present, ngapPdu.InitiatingMessage.ProcedureCode.Value)
		return
	}

	nasPdu = test.GetNasPduSetupRequest(ue, ngapPdu.InitiatingMessage.Value.PDUSessionResourceSetupRequest)
    fmt.Println("Assigned address to UE address ", nasPdu.GmmMessage.DLNASTransport.Ipaddr)
    ueIpaddr := nasPdu.GmmMessage.DLNASTransport.Ipaddr

	// send 14. NGAP-PDU Session Resource Setup Response
    var pduSessionId int64
    pduSessionId = 10
	sendMsg, err = test.GetPDUSessionResourceSetupResponse(pduSessionId, ue.AmfUeNgapId, ue.RanUeNgapId, ranUIpAddr)
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

	// wait 1s
	time.Sleep(1 * time.Second)

	// Send the dummy packet
	// ping IP(tunnel IP) from 60.60.0.2(127.0.0.1) to 60.60.0.20(127.0.0.8)
	gtpHdr, err := hex.DecodeString("32ff00340000000100000000") //tunnel id is hardcoded as  00000001
	if err != nil {
		fmt.Println("Failed to decode  gtpDecode string ")
		return
	}

	icmpData, err := hex.DecodeString("8c870d0000000000101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")
	if err != nil {
		fmt.Println("Failed to decode icmp hexString ")
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
		fmt.Println("ipv4hdr header marshal failed")
		return
	}
	tt := append(gtpHdr, v4HdrBuf...)

	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 12394, Seq: 1,
			Data: icmpData,
		},
	}
	b, err := m.Marshal(nil)
	if err != nil {
		fmt.Println("Failed to marshal gtpu packet ")
		return
	}
	b[2] = 0xaf
	b[3] = 0x88

	upf1, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", upfIpAddr, 2152))
	if err != nil {
		return
	}

	for i := 0; i < 5; i++ {
		_, err = upfConn.WriteToUDP(append(tt, b...), upf1)
		if err != nil {
			fmt.Println("Failed to write gtpu packet ")
		}
		fmt.Println("Sent uplink gtpu packet ")
		time.Sleep(1 * time.Second)
	}

	time.Sleep(100 * time.Second)
	return
}
