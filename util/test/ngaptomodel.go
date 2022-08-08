// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/omec-project/ngap/ngapType"
	"github.com/omec-project/openapi/models"
)

func PDUSessionTypeToModels(ngapPduSessType ngapType.PDUSessionType) (pduSessType models.PduSessionType) {
	switch ngapPduSessType.Value {
	case ngapType.PDUSessionTypePresentIpv4:
		pduSessType = models.PduSessionType_IPV4
	case ngapType.PDUSessionTypePresentIpv6:
		pduSessType = models.PduSessionType_IPV6
	case ngapType.PDUSessionTypePresentIpv4v6:
		pduSessType = models.PduSessionType_IPV4_V6
	case ngapType.PDUSessionTypePresentUnstructured:
		pduSessType = models.PduSessionType_UNSTRUCTURED
	case ngapType.PDUSessionTypePresentEthernet:
		pduSessType = models.PduSessionType_ETHERNET
	}

	return
}
