// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

type iphdr struct {
	vhl   uint8
	tos   uint8
	iplen uint16
	// id    uint16
	off   uint16
	ttl   uint8
	proto uint8
	csum  uint16
	src   [4]byte
	dst   [4]byte
}

type udphdr struct {
	src  uint16
	dst  uint16
	ulen uint16
	csum uint16
}

// pseudo header used for checksum calculation
type pseudohdr struct {
	ipsrc   [4]byte
	ipdst   [4]byte
	zero    uint8
	ipproto uint8
	plen    uint16
}
