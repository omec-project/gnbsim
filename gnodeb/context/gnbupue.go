// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"gnbsim/common"
	"gnbsim/logger"

	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

type GnbUpUe struct {
	// Should IMSI be stored in GnbUe
	PduSessId int64
	DlTeid    int32
	UlTeid    int32
	Snssai    *models.Snssai
	Upf       *GnbUpf
	Gnb       *GNodeB
	// TODO MME details

	// GnbUe writes messages to UE on this channel
	WriteUeChan chan common.InterfaceMessage

	// GnbUe reads messages from all other workers and UE on this channel
	ReadChan chan common.InterfaceMessage

	/* logger */
	Log *logrus.Entry
}

func NewGnbCpUe(ngapId int64, gnb *GNodeB, amf *GnbAmf) *GnbCpUe {
	gnbue := GnbCpUe{}
	gnbue.GnbUeNgapId = ngapId
	gnbue.Amf = amf
	gnbue.Gnb = gnb
	gnbue.ReadChan = make(chan common.InterfaceMessage)
	gnbue.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbUE",
		logger.FieldGnbUeNgapId: ngapId})
	gnbue.Log.Traceln("Context Created")
	return &gnbue
}
