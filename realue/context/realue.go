// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/nas/v2/nasMessage"
	"github.com/omec-project/nas/v2/nasType"
	"github.com/omec-project/nas/v2/security"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/util/milenage"
	"github.com/omec-project/util/ueauth"
	"go.uber.org/zap"
)

// RealUe represents a Real UE
type RealUe struct {
	Log                         *zap.SugaredLogger
	WriteSimUeChan              chan common.InterfaceMessage
	pendingNas                  *nasContext
	nasContexts                 map[nasKeys]nasContext
	nasRetired                  map[[sha256.Size]byte]struct{}
	nasRetiredOrder             [][sha256.Size]byte
	AuthenticationSubs          *models.AuthenticationSubscription
	ReadChan                    chan common.InterfaceMessage
	SNssai                      *models.Snssai
	Plmn                        *models.PlmnId
	PduSessions                 map[int64]*PduSession
	Guti                        string
	Supi                        string
	Dnn                         string
	SeqNum                      string
	Opc                         string
	Key                         string
	Suci                        []uint8
	Kamf                        []uint8
	NgKsi                       models.NgKsi
	WaitGrp                     sync.WaitGroup
	nasContextSeq               uint64
	ULCount                     security.Count
	DLCount                     security.Count
	KnasInt                     [16]uint8
	KnasEnc                     [16]uint8
	NasSecurityEstablished      bool
	IntegrityAlg                uint8
	NasKeysDerived              bool
	CipheringAlg                uint8
	DLCountInUse                bool
	NonSwitchOffDeregPending    bool
	ModificationRequestType     uint8
	OmitModificationRequestType bool
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
	ue.Log = logger.RealUeLog.With(logger.FieldSupi, supi)

	ue.Log.Debugln("created new context")
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

	k, err := hex.DecodeString(authSubs.GetEncPermanentKey())
	if err != nil {
		ue.Log.Fatalf("DecodeString error: %+v", err)
	}

	opc, err := hex.DecodeString(authSubs.GetEncOpcKey())
	if err != nil {
		ue.Log.Fatalf("DecodeString error: %+v", err)
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

	seqNum := hex.EncodeToString(rcvSQN)
	sequenceNumber := models.SequenceNumber{
		Sqn: &seqNum,
	}
	authSubs.SetSequenceNumber(sequenceNumber)

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
		ue.Log.Fatalf("error getting KDF value: %+v", err)
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
		ue.Log.Fatalf("error getting KDF value: %+v", err)
	}
	P0 = []byte(snName)
	Kseaf, err := ueauth.GetKDFValue(Kausf, ueauth.FC_FOR_KSEAF_DERIVATION, P0, ueauth.KDFLen(P0))
	if err != nil {
		ue.Log.Fatalf("error getting KDF value: %+v", err)
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
		ue.Log.Fatalf("error getting KDF value: %+v", err)
	}
}

// nasKeys identifies a 5G NAS security context by its NAS keys.
type nasKeys struct {
	enc, integrity [16]uint8
}

// maxRetainedNasContexts bounds the security contexts remembered besides the one in use, so that
// repeated authentications cannot grow what a UE keeps. When it is exceeded the context that was
// put aside longest ago is retired: its keys are never taken into use again, as they would come
// back with a COUNT of zero (TS 33.501 clause 6.4.5), and only a fingerprint of them is kept. At
// most maxRetiredNasKeys fingerprints are kept, oldest dropped first, which is what bounds the
// memory; a context that old can only come back through an authentication that is that stale.
const (
	maxRetainedNasContexts = 16
	maxRetiredNasKeys      = 1024
)

// nasContext is a 5G NAS security context that is not in use: either keys that wait for the
// SECURITY MODE COMMAND that takes them into use, or a context that was in use before another one.
type nasContext struct {
	ngKsi            models.NgKsi
	keys             nasKeys
	dlCount, ulCount security.Count
	dlCountInUse     bool
	seq              uint64 // when the context was put aside, to retire the oldest first
}

// NewNasSecurityContextPending reports whether derived keys wait for the SECURITY MODE COMMAND
// that takes them into use.
func (ue *RealUe) NewNasSecurityContextPending() bool {
	return ue.pendingNas != nil
}

// PendingNasKeys returns the integrity key of the pending security context, and the downlink COUNT
// state a SECURITY MODE COMMAND for it is checked against: the COUNT last accepted in it and
// whether one was. A context that has never been in use starts with no COUNT accepted.
func (ue *RealUe) PendingNasKeys() (integrity [16]uint8, dlCount security.Count, dlCountInUse, ok bool) {
	if ue.pendingNas == nil {
		return integrity, dlCount, false, false
	}
	return ue.pendingNas.keys.integrity, ue.pendingNas.dlCount, ue.pendingNas.dlCountInUse, true
}

// ActivatePendingNasContext takes the pending security context into use, once the SECURITY MODE
// COMMAND for it has passed its integrity check at dlCount. The context that was in use is kept, so
// that it can be put back with its COUNTs if the network moves back to it. The uplink COUNT starts
// over for a context that has never been in use (TS 24.501 clause 5.4.2.3), and is otherwise the
// one the context had reached. ngKsi is the key set identifier of the new context as given by the
// authenticated SECURITY MODE COMMAND; it changes with the keys, not before: until then requests
// keep naming the context that protects them. Without a pending context it does nothing.
func (ue *RealUe) ActivatePendingNasContext(dlCount security.Count, ngKsi models.NgKsi) {
	next := ue.pendingNas
	if next == nil {
		return
	}
	if ue.NasKeysDerived {
		if ue.nasContexts == nil {
			ue.nasContexts = make(map[nasKeys]nasContext)
		}
		current := nasKeys{enc: ue.KnasEnc, integrity: ue.KnasInt}
		ue.nasContextSeq++
		ue.nasContexts[current] = nasContext{
			keys: current, ngKsi: ue.NgKsi, dlCount: ue.DLCount, ulCount: ue.ULCount, dlCountInUse: ue.DLCountInUse,
			seq: ue.nasContextSeq,
		}
		ue.retireOldestNasContexts()
	}
	delete(ue.nasContexts, next.keys)
	ue.KnasEnc, ue.KnasInt = next.keys.enc, next.keys.integrity
	ue.NgKsi = ngKsi
	ue.NasKeysDerived = true
	ue.DLCount, ue.ULCount = dlCount, next.ulCount
	ue.DLCountInUse = true
	ue.pendingNas = nil
}

func fingerprint(k nasKeys) [sha256.Size]byte {
	return sha256.Sum256(append(append([]byte(nil), k.enc[:]...), k.integrity[:]...))
}

// retireOldestNasContexts drops the contexts beyond maxRetainedNasContexts, oldest first, keeping a
// fingerprint of their keys so that they cannot be staged again.
func (ue *RealUe) retireOldestNasContexts() {
	for len(ue.nasContexts) > maxRetainedNasContexts {
		var oldest nasContext
		first := true
		for _, c := range ue.nasContexts {
			if first || c.seq < oldest.seq {
				oldest, first = c, false
			}
		}
		delete(ue.nasContexts, oldest.keys)
		if ue.nasRetired == nil {
			ue.nasRetired = make(map[[sha256.Size]byte]struct{})
		}
		fp := fingerprint(oldest.keys)
		ue.nasRetired[fp] = struct{}{}
		ue.nasRetiredOrder = append(ue.nasRetiredOrder, fp)
		if len(ue.nasRetiredOrder) > maxRetiredNasKeys {
			delete(ue.nasRetired, ue.nasRetiredOrder[0])
			ue.nasRetiredOrder = ue.nasRetiredOrder[1:]
		}
	}
}

// Algorithm key Derivation function defined in TS 33.501 Annex A.9. The keys are staged, not
// taken into use. The key set identifier is not kept with them: the AUTHENTICATION REQUEST that
// carries it is not integrity protected, so it is taken from the SECURITY MODE COMMAND instead.
func (ue *RealUe) DerivateAlgKey() {
	// Security Key
	P0 := []byte{security.NNASEncAlg}
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{ue.CipheringAlg}
	L1 := ueauth.KDFLen(P1)

	kenc, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		ue.Log.Fatalf("error getting KDF value: %+v", err)
	}
	var newKnasEnc, newKnasInt [16]uint8
	copy(newKnasEnc[:], kenc[16:32])

	// Integrity Key
	P0 = []byte{security.NNASIntAlg}
	L0 = ueauth.KDFLen(P0)
	P1 = []byte{ue.IntegrityAlg}
	L1 = ueauth.KDFLen(P1)

	kint, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		ue.Log.Fatalf("error getting KDF value: %+v", err)
	}
	copy(newKnasInt[:], kint[16:32])

	// The keys are not taken into use here: that takes a SECURITY MODE COMMAND that passes its
	// integrity check under them (TS 24.501 clause 5.4.2.3). AUTHENTICATION REQUEST is not
	// integrity protected, so until then the context in use keeps verifying and protecting
	// messages, and a single pending context is kept so that a stream of requests cannot grow
	// what is remembered.
	derived := nasKeys{enc: newKnasEnc, integrity: newKnasInt}
	if ue.NasKeysDerived && derived == (nasKeys{enc: ue.KnasEnc, integrity: ue.KnasInt}) {
		// a repeated request for the context in use is not a new context. It is the latest
		// authentication, though, so keys staged by an earlier one are stale and not kept.
		ue.pendingNas = nil
		return
	}
	if ue.pendingNas != nil && ue.pendingNas.keys == derived {
		return
	}
	// The keys of a context that was in use before come back with its COUNT state, or a recorded
	// SECURITY MODE COMMAND of it would verify again at its original COUNT.
	if state, ok := ue.nasContexts[derived]; ok {
		ue.pendingNas = &state
		return
	}
	// Keys of a retired context would restart their COUNTs. This authentication is the latest, so
	// keys staged by an earlier one are stale and not kept either.
	if _, retired := ue.nasRetired[fingerprint(derived)]; retired {
		ue.Log.Warnln("ignoring the keys of a retired NAS security context")
		ue.pendingNas = nil
		return
	}
	ue.pendingNas = &nasContext{keys: derived}
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
	ctx.Log.Infoln("fetching PDU Session for pduSessId:", pduSessId)
	val, ok := ctx.PduSessions[pduSessId]
	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("key not present: %v", pduSessId)
	}
}

// AddPduSession adds the PduSession instance corresponding to provided PDU Sess ID
func (ctx *RealUe) AddPduSession(pduSessId int64, pduSess *PduSession) {
	ctx.Log.Infoln("adding new PDU Session for PDU Sess ID:", pduSessId)
	ctx.PduSessions[pduSessId] = pduSess
}

// OnlyPduSession returns the UE's single established PDU session.
//
// The simulator establishes one session per UE. Returning it rather than assuming its identity
// keeps a caller correct if that ever stops being true, and gives a clear error instead of a
// silent zero when no session exists.
func (ue *RealUe) OnlyPduSession() (int64, *PduSession, error) {
	if len(ue.PduSessions) != 1 {
		return 0, nil, fmt.Errorf("expected exactly one PDU session, found %d", len(ue.PduSessions))
	}
	for id, sess := range ue.PduSessions {
		return id, sess, nil
	}
	return 0, nil, fmt.Errorf("unreachable")
}
