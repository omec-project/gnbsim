// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package common

type ProcedureType uint8

const (
	REGISTRATION_PROCEDURE ProcedureType = 1 + iota
	PDU_SESSION_ESTABLISHMENT_PROCEDURE
	USER_DATA_PKT_GENERATION_PROCEDURE
	UE_INITIATED_DEREGISTRATION_PROCEDURE
)
