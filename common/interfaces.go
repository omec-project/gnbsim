// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package common

type InterfaceType uint8

const (
	// Application defined interfaces
	PROFILE_SIMUE_INTERFACE InterfaceType = 1 + iota

	// Network interfaces
	UU_INTERFACE
	N2_INTERFACE
	N3_INTERFACE
)
