// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"net"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"

	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

/* PduSession represents a PDU Session in Real UE. It listens for DL user data
 * packets from the gNB and also writes UL packets to gNB on the command of
 * Real UE control plane
 */
type PduSession struct {
	/* Number of UL data packets to be transmitted as requested by Sim UE*/
	SscMode          uint8
	PktCount         int
	PduSessId        int64
	Snssai           models.Snssai
	PduSessType      models.PduSessionType
	PduAddress       net.IP
	SeqNum           int
	ReqDataPktCount  int
	ReqDataPktInt    uint32
	DefaultAs        string
	TxDataPktCount   int
	RxDataPktCount   int
	LastDataPktRecvd bool
	// Inidicates that a Go routine already exists for this PDU Session
	Launched bool
	/* uplink packets are written to gNB UE user plane context on this channel */
	WriteGnbChan chan common.InterfaceMessage

	/* command replies are written to RealUE over this channel */
	WriteUeChan chan common.InterfaceMessage

	/* Downlink packets from gNB UE user plane context are read over this channel */
	ReadDlChan chan common.InterfaceMessage

	// commands from RealUE control plane are read on this channel
	ReadCmdChan chan common.InterfaceMessage

	/* logger */
	Log *logrus.Entry
}

func NewPduSession(realUe *RealUe, pduSessId int64) *PduSession {
	pduSess := PduSession{}
	pduSess.PduSessId = pduSessId
	pduSess.ReadDlChan = make(chan common.InterfaceMessage, 10)
	pduSess.ReadCmdChan = make(chan common.InterfaceMessage, 10)
	pduSess.Log = realUe.Log.WithFields(logrus.Fields{"subcategory": "PduSession",
		logger.FieldPduSessId: pduSessId})
	pduSess.Log.Traceln("Pdu Session Created")
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
