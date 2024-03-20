// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"encoding/hex"
	"regexp"

	"github.com/calee0219/fatal"
	"github.com/omec-project/CommonConsumerTestData/UDR/TestRegistrationProcedure"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/milenage"
	"github.com/omec-project/util/ueauth"
	"golang.org/x/net/ipv4"
)

type RanUeContext struct {
	Supi               string
	AuthenticationSubs models.AuthenticationSubscription
	Kamf               []uint8
	RanUeNgapId        int64
	AmfUeNgapId        int64
	ULCount            security.Count
	DLCount            security.Count
	CipheringAlg       uint8
	IntegrityAlg       uint8
	KnasEnc            [16]uint8
	KnasInt            [16]uint8
}

func CalculateIpv4HeaderChecksum(hdr *ipv4.Header) uint32 {
	var Checksum uint32
	Checksum += uint32((hdr.Version<<4|(20>>2&0x0f))<<8 | hdr.TOS)
	Checksum += uint32(hdr.TotalLen)
	Checksum += uint32(hdr.ID)
	Checksum += uint32((hdr.FragOff & 0x1fff) | (int(hdr.Flags) << 13))
	Checksum += uint32((hdr.TTL << 8) | (hdr.Protocol))

	src := hdr.Src.To4()
	Checksum += uint32(src[0])<<8 | uint32(src[1])
	Checksum += uint32(src[2])<<8 | uint32(src[3])
	dst := hdr.Dst.To4()
	Checksum += uint32(dst[0])<<8 | uint32(dst[1])
	Checksum += uint32(dst[2])<<8 | uint32(dst[3])
	return ^(Checksum&0xffff0000>>16 + Checksum&0xffff)
}

func GetAuthSubscription(k, opc, op, seqNum string) *models.AuthenticationSubscription {
	var authSubs models.AuthenticationSubscription
	authSubs.PermanentKey = &models.PermanentKey{
		PermanentKeyValue: k,
	}
	authSubs.Opc = &models.Opc{
		OpcValue: opc,
	}
	authSubs.Milenage = &models.Milenage{
		Op: &models.Op{
			OpValue: op,
		},
	}
	authSubs.AuthenticationManagementField = "8000"

	authSubs.SequenceNumber = seqNum
	authSubs.AuthenticationMethod = models.AuthMethod__5_G_AKA
	return &authSubs
}

func GetAccessAndMobilitySubscriptionData() (amData models.AccessAndMobilitySubscriptionData) {
	return TestRegistrationProcedure.TestAmDataTable[TestRegistrationProcedure.FREE5GC_CASE]
}

func GetSmfSelectionSubscriptionData() (smfSelData models.SmfSelectionSubscriptionData) {
	return TestRegistrationProcedure.TestSmfSelDataTable[TestRegistrationProcedure.FREE5GC_CASE]
}

func GetSessionManagementSubscriptionData() (smfSelData []models.SessionManagementSubscriptionData) {
	return TestRegistrationProcedure.TestSmSelDataTable[TestRegistrationProcedure.FREE5GC_CASE]
}

func GetAmPolicyData() (amPolicyData models.AmPolicyData) {
	return TestRegistrationProcedure.TestAmPolicyDataTable[TestRegistrationProcedure.FREE5GC_CASE]
}

func GetSmPolicyData() (smPolicyData models.SmPolicyData) {
	return TestRegistrationProcedure.TestSmPolicyDataTable[TestRegistrationProcedure.FREE5GC_CASE]
}

func NewRanUeContext(supi string, ranUeNgapId int64, cipheringAlg, integrityAlg uint8) *RanUeContext {
	ue := RanUeContext{}
	ue.RanUeNgapId = ranUeNgapId
	ue.Supi = supi
	ue.CipheringAlg = cipheringAlg
	ue.IntegrityAlg = integrityAlg
	return &ue
}

func (ue *RanUeContext) DeriveRESstarAndSetKey(
	authSubs models.AuthenticationSubscription, rand []byte, snName string,
) []byte {
	sqn, err := hex.DecodeString(authSubs.SequenceNumber)
	if err != nil {
		fatal.Fatalf("DecodeString error: %+v", err)
	}

	amf, err := hex.DecodeString(authSubs.AuthenticationManagementField)
	if err != nil {
		fatal.Fatalf("DecodeString error: %+v", err)
	}

	// Run milenage
	macA, macS := make([]byte, 8), make([]byte, 8)
	ck, ik := make([]byte, 16), make([]byte, 16)
	res := make([]byte, 8)
	ak, akStar := make([]byte, 6), make([]byte, 6)

	opc := make([]byte, 16)
	_ = opc
	k, err := hex.DecodeString(authSubs.PermanentKey.PermanentKeyValue)
	if err != nil {
		fatal.Fatalf("DecodeString error: %+v", err)
	}

	if authSubs.Opc.OpcValue == "" {
		opStr := authSubs.Milenage.Op.OpValue
		var op []byte
		op, err = hex.DecodeString(opStr)
		if err != nil {
			fatal.Fatalf("DecodeString error: %+v", err)
		}

		opc, err = milenage.GenerateOPC(k, op)
		if err != nil {
			fatal.Fatalf("milenage GenerateOPC error: %+v", err)
		}
	} else {
		opc, err = hex.DecodeString(authSubs.Opc.OpcValue)
		if err != nil {
			fatal.Fatalf("DecodeString error: %+v", err)
		}
	}

	// Generate MAC_A, MAC_S
	err = milenage.F1(opc, k, rand, sqn, amf, macA, macS)
	if err != nil {
		fatal.Fatalf("regexp Compile error: %+v", err)
	}

	// Generate RES, CK, IK, AK, AKstar
	err = milenage.F2345(opc, k, rand, res, ck, ik, ak, akStar)
	if err != nil {
		fatal.Fatalf("regexp Compile error: %+v", err)
	}

	// derive RES*
	key := append(ck, ik...)
	FC := ueauth.FC_FOR_RES_STAR_XRES_STAR_DERIVATION
	P0 := []byte(snName)
	P1 := rand
	P2 := res

	ue.DerivateKamf(key, snName, sqn, ak)
	ue.DerivateAlgKey()
	kdfVal_for_resStar, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1), P2, ueauth.KDFLen(P2))
	if err != nil {
		fatal.Fatalf("Error getting KDF value: %+v", err)
	}
	return kdfVal_for_resStar[len(kdfVal_for_resStar)/2:]
}

func (ue *RanUeContext) DerivateKamf(key []byte, snName string, SQN, AK []byte) {
	FC := ueauth.FC_FOR_KAUSF_DERIVATION
	P0 := []byte(snName)
	SQNxorAK := make([]byte, 6)
	for i := 0; i < len(SQN); i++ {
		SQNxorAK[i] = SQN[i] ^ AK[i]
	}
	P1 := SQNxorAK
	Kausf, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1))
	if err != nil {
		fatal.Fatalf("Error getting KDF value: %+v", err)
	}
	P0 = []byte(snName)
	Kseaf, err := ueauth.GetKDFValue(Kausf, ueauth.FC_FOR_KSEAF_DERIVATION, P0, ueauth.KDFLen(P0))
	if err != nil {
		fatal.Fatalf("Error getting KDF value: %+v", err)
	}

	supiRegexp, err := regexp.Compile("(?:imsi|supi)-([0-9]{5,15})")
	if err != nil {
		fatal.Fatalf("regexp Compile error: %+v", err)
	}
	groups := supiRegexp.FindStringSubmatch(ue.Supi)

	P0 = []byte(groups[1])
	L0 := ueauth.KDFLen(P0)
	P1 = []byte{0x00, 0x00}
	L1 := ueauth.KDFLen(P1)

	ue.Kamf, err = ueauth.GetKDFValue(Kseaf, ueauth.FC_FOR_KAMF_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		fatal.Fatalf("Error getting KDF value: %+v", err)
	}
}

// Algorithm key Derivation function defined in TS 33.501 Annex A.9
func (ue *RanUeContext) DerivateAlgKey() {
	// Security Key
	P0 := []byte{security.NNASEncAlg}
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{ue.CipheringAlg}
	L1 := ueauth.KDFLen(P1)

	kenc, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		fatal.Fatalf("Error getting KDF value: %+v", err)
	}
	copy(ue.KnasEnc[:], kenc[16:32])

	// Integrity Key
	P0 = []byte{security.NNASIntAlg}
	L0 = ueauth.KDFLen(P0)
	P1 = []byte{ue.IntegrityAlg}
	L1 = ueauth.KDFLen(P1)

	kint, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		fatal.Fatalf("Error getting KDF value: %+v", err)
	}
	copy(ue.KnasInt[:], kint[16:32])
}

func (ue *RanUeContext) GetUESecurityCapability() (UESecurityCapability *nasType.UESecurityCapability) {
	UESecurityCapability = &nasType.UESecurityCapability{
		Iei:    nasMessage.RegistrationRequestUESecurityCapabilityType,
		Len:    2,
		Buffer: []uint8{0x00, 0x00},
	}
	switch ue.CipheringAlg {
	case security.AlgCiphering128NEA0:
		UESecurityCapability.SetEA0_5G(1)
	case security.AlgCiphering128NEA1:
		UESecurityCapability.SetEA1_128_5G(1)
	case security.AlgCiphering128NEA2:
		UESecurityCapability.SetEA2_128_5G(1)
	case security.AlgCiphering128NEA3:
		UESecurityCapability.SetEA3_128_5G(1)
	}

	switch ue.IntegrityAlg {
	case security.AlgIntegrity128NIA0:
		UESecurityCapability.SetIA0_5G(1)
	case security.AlgIntegrity128NIA1:
		UESecurityCapability.SetIA1_128_5G(1)
	case security.AlgIntegrity128NIA2:
		UESecurityCapability.SetIA2_128_5G(1)
	case security.AlgIntegrity128NIA3:
		UESecurityCapability.SetIA3_128_5G(1)
	}

	return
}

func (ue *RanUeContext) Get5GMMCapability() (capability5GMM *nasType.Capability5GMM) {
	return &nasType.Capability5GMM{
		Iei:   nasMessage.RegistrationRequestCapability5GMMType,
		Len:   1,
		Octet: [13]uint8{0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
}
