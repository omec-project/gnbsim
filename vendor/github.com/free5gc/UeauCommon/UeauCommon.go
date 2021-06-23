package UeauCommon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"log"
)

const (
	FC_FOR_CK_PRIME_IK_PRIME_DERIVATION  = "20"
	FC_FOR_KSEAF_DERIVATION              = "6C"
	FC_FOR_RES_STAR_XRES_STAR_DERIVATION = "6B"
	FC_FOR_KAUSF_DERIVATION              = "6A"
	FC_FOR_KAMF_DERIVATION               = "6D"
	FC_FOR_KGNB_KN3IWF_DERIVATION        = "6E"
	FC_FOR_NH_DERIVATION                 = "6F"
	FC_FOR_ALGORITHM_KEY_DERIVATION      = "69"
)

func KDFLen(input []byte) []byte {
	var r = make([]byte, 2)
	binary.BigEndian.PutUint16(r, uint16(len(input)))
	return r
}

// This function implements the KDF defined in TS.33220 cluase B.2.0.
//
// For P0-Pn, the ones that will be used directly as a string (e.g. "WLAN") should be type-casted by []byte(),
// and the ones originially in hex (e.g. "bb52e91c747a") should be converted by using hex.DecodeString().
//
// For L0-Ln, use KDFLen() function to calculate them (e.g. KDFLen(P0)).
func GetKDFValue(key []byte, FC string, param ...[]byte) []byte {
	kdf := hmac.New(sha256.New, key)

	var S []byte
	if STmp, err := hex.DecodeString(string(FC)); err != nil {
		log.Printf("Hex decode failed: %+v", err)
	} else {
		S = STmp
	}

	for _, p := range param {
		S = append(S, p...)
	}

	if _, err := kdf.Write(S); err != nil {
		log.Printf("KDF write failed: %+v", err)
	}
	sum := kdf.Sum(nil)
	return sum
}
