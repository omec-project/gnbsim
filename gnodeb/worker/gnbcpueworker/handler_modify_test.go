// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package gnbcpueworker

import (
	"testing"

	"github.com/omec-project/ngap/v2/ngapType"
)

// releaseTransfer builds a modify request transfer carrying a QoS Flow to Release List, which is
// the shape the SMF sends when it withdraws a flow: that request names nothing else.
func releaseTransfer(qfis ...int64) *ngapType.PDUSessionResourceModifyRequestTransfer {
	list := &ngapType.QosFlowListWithCause{}
	for _, qfi := range qfis {
		list.List = append(list.List, ngapType.QosFlowWithCauseItem{
			QosFlowIdentifier: ngapType.QosFlowIdentifier{Value: qfi},
		})
	}

	transfer := &ngapType.PDUSessionResourceModifyRequestTransfer{}
	transfer.ProtocolIEs.List = append(transfer.ProtocolIEs.List,
		ngapType.PDUSessionResourceModifyRequestTransferIEs{
			Id: ngapType.ProtocolIEID{Value: ngapType.ProtocolIEIDQosFlowToReleaseList},
			Value: ngapType.PDUSessionResourceModifyRequestTransferIEsValue{
				Present:              ngapType.PDUSessionResourceModifyRequestTransferIEsPresentQosFlowToReleaseList,
				QosFlowToReleaseList: list,
			},
		})
	return transfer
}

func TestReleasedQfis(t *testing.T) {
	tests := []struct {
		name     string
		transfer *ngapType.PDUSessionResourceModifyRequestTransfer
		want     []int64
	}{
		{
			name:     "a release-only request names the flows to drop",
			transfer: releaseTransfer(2, 3),
			want:     []int64{2, 3},
		},
		{
			name:     "a request with no release list releases nothing",
			transfer: &ngapType.PDUSessionResourceModifyRequestTransfer{},
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := releasedQfis(tc.transfer)
			if len(got) != len(tc.want) {
				t.Fatalf("releasedQfis() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("releasedQfis()[%d] = %d, want %d", i, got[i], tc.want[i])
				}
			}
		})
	}
}
