// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package transportcommon

type Transport interface {
	Init() error
	ConnectToPeer(peer TransportPeer) error
	SendToPeerBlock(peer TransportPeer, pkt []byte) ([]byte, error)
	SendToPeer(peer TransportPeer, pkt []byte) (err error)
	ReceiveFromPeer(peer TransportPeer)
	CheckTransportParam(peer TransportPeer, pkt []byte) error
}
