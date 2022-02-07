// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transportcommon

type TransportPeer interface {
	GetIpAddr() string
	GetPort() int
}
