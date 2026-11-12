// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"net"
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/openapi/v2/models"
	"go.uber.org/zap"
)

/* PduSession represents a PDU Session in Real UE. It listens for DL user data
 * packets from the gNB and also writes UL packets to gNB on the command of
 * Real UE control plane
 */
type PduSession struct {
	Snssai           models.Snssai
	WriteUeChan      chan common.InterfaceMessage
	ReadDlChan       chan common.InterfaceMessage
	ReadCmdChan      chan common.InterfaceMessage
	Log              *zap.SugaredLogger
	GenStop          chan struct{}
	WriteGnbChan     chan common.InterfaceMessage
	DefaultAs        string
	PduSessType      models.PduSessionType
	PduAddress       net.IP
	GenWg            sync.WaitGroup
	PktCount         int
	SeqNum           int
	ReqDataPktCount  int
	ReqDataPktInt    int
	TxDataPktCount   int
	RxDataPktCount   int
	PduSessId        int64
	SscMode          uint8
	LastDataPktRecvd bool
	Launched         bool
	PendingPTI       uint8
}

func NewPduSession(realUe *RealUe, pduSessId int64) *PduSession {
	pduSess := PduSession{}
	pduSess.PduSessId = pduSessId
	pduSess.ReadDlChan = make(chan common.InterfaceMessage, 10)
	pduSess.ReadCmdChan = make(chan common.InterfaceMessage, 10)
	pduSess.Log = realUe.Log.With("subcategory", "PduSession", logger.FieldPduSessId, pduSessId)
	pduSess.Log.Debugln("pdu session created")
	return &pduSess
}

func (pduSess *PduSession) GetNextSeqNum() int {
	pduSess.SeqNum++
	/* Allowing sequence number to always start from 1 */
	if pduSess.SeqNum <= 0 {
		pduSess.SeqNum = 1
	}
	return pduSess.SeqNum
}
