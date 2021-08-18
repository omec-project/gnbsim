// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package interfacecommon

type EventType uint16

// UE related events
const (
	UE_REG_REQ              EventType = iota
	UE_UPLINK_NAS_TRANSPORT EventType = iota
)

// GNodeB related events
const (
	GNB_DOWNLINK_NAS_TRANSPORT EventType = iota
)

// AMF related events
const (
	AMF_DOWNLINK_NAS_TRANSPORT        EventType = iota
	AMF_INITIAL_CONTEXT_SETUP_REQUEST EventType = iota
)
