// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"gnbsim/common"
	"gnbsim/logger"
	"net"

	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

/* PduSession represents a PDU Session in Real UE. It listens for DL user data
 * packets from the gNB and also writes UL packets to gNB on the command of
 * Real UE control plane
 */
type PduSession struct {
	/* Number of UL data packets to be transmitted as requested by Sim UE*/
	SscMode     uint8
	PktCount    int
	PduSessId   uint64
	Snssai      models.Snssai
	PduSessType models.PduSessionType
	PduAddress  net.IP

	/* uplink packets are written to gNB UE user plane context on this channel */
	WriteGnbChan chan common.InterfaceMessage

	/* Downlink packets from gNB UE user plane context are read over this channel */
	ReadDlChan chan common.InterfaceMessage

	// commands from RealUE control plane are read on this channel
	ReadCmdChan chan common.InterfaceMessage

	/* logger */
	Log *logrus.Entry
}

func NewPduSession(realUe *RealUe, pduSessId uint64) *PduSession {
	pduSess := PduSession{}
	pduSess.ReadDlChan = make(chan common.InterfaceMessage, 10)
	pduSess.ReadCmdChan = make(chan common.InterfaceMessage)
	pduSess.Log = logger.RealUeLog.WithFields(logrus.Fields{"subcategory": "PduSession",
		logger.FieldPduSessId: pduSessId})
	pduSess.Log.Traceln("Pdu Session Created")
	return &pduSess
}
