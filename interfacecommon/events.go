// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package interfacecommon

type EventType uint8

// General UE events
const (
	UE_CONNECTION_REQ EventType = iota
	UE_UPLINK_NAS_TRANSPORT
)

// Following events are numbered same as the NAS Message types in section 9.7 of
// 3GPP TS 24.501

// 5GS Mobility Management UE events.
const UE_5GS_MOBILITY_MANAGEMENT_EVENTS EventType = 64
const (
	_ EventType = UE_5GS_MOBILITY_MANAGEMENT_EVENTS + iota
	UE_REG_REQUEST
	UE_REG_ACCEPT //66
	UE_REG_COMPLETE
	UE_REG_REJECT
	UE_DEREG_REQUEST_ORIG
	UE_DEREG_ACCEPT_ORIG
	UE_DEREG_REQUEST_TERM
	UE_DEREG_ACCEPT_TERM //72

	UE_SERVICE_REQUEST = UE_5GS_MOBILITY_MANAGEMENT_EVENTS + 3 + iota //76
	UE_SERVICE_REJECT
	UE_SERVICE_ACCEPT

	UE_AUTH_REQUEST = UE_5GS_MOBILITY_MANAGEMENT_EVENTS + 10 + iota //86
	UE_AUTH_RESPONSE
	UE_AUTH_REJECT
	UE_AUTH_FAILURE
	UE_AUTH_RESULT
	UE_ID_REQUEST
	UE_ID_RESPONSE
	UE_SEC_MOD_COMMAND
	UE_SEC_MOD_COMPLETE
	UE_SEC_MOD_REJECT //95
)

// GNodeB related events
const (
	GNB_DOWNLINK_NAS_TRANSPORT EventType = iota
)

// AMF related events
const (
	AMF_DOWNLINK_NAS_TRANSPORT EventType = iota
	AMF_INITIAL_CONTEXT_SETUP_REQUEST
)
