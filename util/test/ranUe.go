// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/omec-project/nas/security"
	"github.com/omec-project/openapi/models"
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
