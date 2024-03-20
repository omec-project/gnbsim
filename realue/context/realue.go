// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/nas/nasMessage"
	"github.com/omec-project/nas/nasType"
	"github.com/omec-project/nas/security"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/milenage"
	"github.com/omec-project/util/ueauth"
	"github.com/sirupsen/logrus"
)

// RealUe represents a Real UE
type RealUe struct {
	Supi               string
	Guti               string
	Key                string
	Opc                string
	SeqNum             string
	Dnn                string
	SNssai             *models.Snssai
	AuthenticationSubs *models.AuthenticationSubscription
	Plmn               *models.PlmnId
	Log                *logrus.Entry

	// RealUe writes messages to SimUE on this channel
	WriteSimUeChan chan common.InterfaceMessage

	// RealUe reads messages from SimUE on this channel
	ReadChan chan common.InterfaceMessage

	PduSessions  map[int64]*PduSession
	Kamf         []uint8
	Suci         []uint8
	NgKsi        models.NgKsi
	WaitGrp      sync.WaitGroup
	ULCount      security.Count
	DLCount      security.Count
	CipheringAlg uint8
	IntegrityAlg uint8
	KnasEnc      [16]uint8
	KnasInt      [16]uint8
}

func NewRealUe(supi string, cipheringAlg, integrityAlg uint8,
	simuechan chan common.InterfaceMessage, plmnid *models.PlmnId,
	key string, opc string, seqNum string, Dnn string, SNssai *models.Snssai,
) *RealUe {
	ue := RealUe{}
	ue.Supi = supi
	ue.CipheringAlg = cipheringAlg
	ue.IntegrityAlg = integrityAlg
	ue.Key = key
	ue.Opc = opc
	ue.SeqNum = seqNum
	ue.Dnn = Dnn
	ue.SNssai = SNssai
	ue.Plmn = plmnid
	ue.WriteSimUeChan = simuechan
	ue.PduSessions = make(map[int64]*PduSession)
	ue.ReadChan = make(chan common.InterfaceMessage, 5)
	ue.Log = logger.RealUeLog.WithField(logger.FieldSupi, supi)

	ue.Log.Traceln("Created new context")
	return &ue
}

func (ue *RealUe) GetUESecurityCapability() (UESecurityCapability *nasType.UESecurityCapability) {
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

func (ue *RealUe) DeriveRESstarAndSetKey(
	autn, rand []byte, snName string,
) []byte {
	authSubs := ue.AuthenticationSubs

	// Run milenage
	macA, macS := make([]byte, 8), make([]byte, 8)
	ck, ik := make([]byte, 16), make([]byte, 16)
	res := make([]byte, 8)
	ak, akStar := make([]byte, 6), make([]byte, 6)

	opc := make([]byte, 16)
	_ = opc
	k, err := hex.DecodeString(authSubs.PermanentKey.PermanentKeyValue)
	if err != nil {
		ue.Log.Fatalf("DecodeString error: %+v", err)
	}

	if authSubs.Opc.OpcValue == "" {
		opStr := authSubs.Milenage.Op.OpValue
		var op []byte
		op, err = hex.DecodeString(opStr)
		if err != nil {
			ue.Log.Fatalf("DecodeString error: %+v", err)
		}

		opc, err = milenage.GenerateOPC(k, op)
		if err != nil {
			ue.Log.Fatalf("milenage GenerateOPC error: %+v", err)
		}
	} else {
		opc, err = hex.DecodeString(authSubs.Opc.OpcValue)
		if err != nil {
			ue.Log.Fatalf("DecodeString error: %+v", err)
		}
	}

	// Generate RES, CK, IK, AK, AKstar
	err = milenage.F2345(opc, k, rand, res, ck, ik, ak, akStar)
	if err != nil {
		ue.Log.Fatalf("regexp Compile error: %+v", err)
	}

	rcvSQN := make([]byte, 6)

	// Todo : check what to do with the separation bit of the AMF field
	rcvAMF := autn[6:8]

	for i := 0; i < 6; i++ {
		rcvSQN[i] = ak[i] ^ autn[i]
	}

	authSubs.SequenceNumber = hex.EncodeToString(rcvSQN)

	// Todo : Figure 9 of 33.102 shows that we can use the SQN received from the
	// network to calculate XMAC which we can then compare with the received MAC
	// value in the Authentication Request(AUTN IE) to authenticate the network

	// Generate MAC_A, MAC_S
	err = milenage.F1(opc, k, rand, rcvSQN, rcvAMF, macA, macS)
	if err != nil {
		ue.Log.Fatalf("regexp Compile error: %+v", err)
	}

	// derive RES*
	key := append(ck, ik...)
	FC := ueauth.FC_FOR_RES_STAR_XRES_STAR_DERIVATION
	P0 := []byte(snName)
	P1 := rand
	P2 := res

	ue.DerivateKamf(key, snName, rcvSQN, ak)
	ue.DerivateAlgKey()
	kdfVal_for_resStar, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1), P2, ueauth.KDFLen(P2))
	if err != nil {
		ue.Log.Fatalf("Error getting KDF value: %+v", err)
	}
	return kdfVal_for_resStar[len(kdfVal_for_resStar)/2:]
}

func (ue *RealUe) DerivateKamf(key []byte, snName string, SQN, AK []byte) {
	FC := ueauth.FC_FOR_KAUSF_DERIVATION
	P0 := []byte(snName)
	SQNxorAK := make([]byte, 6)
	for i := 0; i < len(SQN); i++ {
		SQNxorAK[i] = SQN[i] ^ AK[i]
	}
	P1 := SQNxorAK
	Kausf, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1))
	if err != nil {
		ue.Log.Fatalf("Error getting KDF value: %+v", err)
	}
	P0 = []byte(snName)
	Kseaf, err := ueauth.GetKDFValue(Kausf, ueauth.FC_FOR_KSEAF_DERIVATION, P0, ueauth.KDFLen(P0))
	if err != nil {
		ue.Log.Fatalf("Error getting KDF value: %+v", err)
	}

	supiRegexp, err := regexp.Compile("(?:imsi|supi)-([0-9]{5,15})")
	if err != nil {
		ue.Log.Fatalf("regexp Compile error: %+v", err)
	}
	groups := supiRegexp.FindStringSubmatch(ue.Supi)

	P0 = []byte(groups[1])
	L0 := ueauth.KDFLen(P0)
	P1 = []byte{0x00, 0x00}
	L1 := ueauth.KDFLen(P1)

	ue.Kamf, err = ueauth.GetKDFValue(Kseaf, ueauth.FC_FOR_KAMF_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		ue.Log.Fatalf("Error getting KDF value: %+v", err)
	}
}

// Algorithm key Derivation function defined in TS 33.501 Annex A.9
func (ue *RealUe) DerivateAlgKey() {
	// Security Key
	P0 := []byte{security.NNASEncAlg}
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{ue.CipheringAlg}
	L1 := ueauth.KDFLen(P1)

	kenc, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		ue.Log.Fatalf("Error getting KDF value: %+v", err)
	}
	copy(ue.KnasEnc[:], kenc[16:32])

	// Integrity Key
	P0 = []byte{security.NNASIntAlg}
	L0 = ueauth.KDFLen(P0)
	P1 = []byte{ue.IntegrityAlg}
	L1 = ueauth.KDFLen(P1)

	kint, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		ue.Log.Fatalf("Error getting KDF value: %+v", err)
	}
	copy(ue.KnasInt[:], kint[16:32])
}

func (ue *RealUe) Get5GMMCapability() (capability5GMM *nasType.Capability5GMM) {
	return &nasType.Capability5GMM{
		Iei:   nasMessage.RegistrationRequestCapability5GMMType,
		Len:   1,
		Octet: [13]uint8{0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
}

// GetPduSession returns the PduSession instance corresponding to provided PDU Sess ID
func (ctx *RealUe) GetPduSession(pduSessId int64) (*PduSession, error) {
	ctx.Log.Infoln("Fetching PDU Session for pduSessId:", pduSessId)
	val, ok := ctx.PduSessions[pduSessId]
	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("key not present: %v", pduSessId)
	}
}

// AddPduSession adds the PduSession instance corresponding to provided PDU Sess ID
func (ctx *RealUe) AddPduSession(pduSessId int64, pduSess *PduSession) {
	ctx.Log.Infoln("Adding new PDU Session for PDU Sess ID:", pduSessId)
	ctx.PduSessions[pduSessId] = pduSess
}
