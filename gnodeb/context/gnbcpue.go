// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/gnbsim/logger"

	"github.com/sirupsen/logrus"
)

type GnbCpUe struct {
	Supi        string
	GnbUeNgapId int64
	AmfUeNgapId int64
	Amf         *GnbAmf
	Gnb         *GNodeB

	// TODO: Sync map is not needed as it is handled single threaded
	GnbUpUes sync.Map

	WaitGrp sync.WaitGroup

	// GnbCpUe writes messages to UE on this channel
	WriteUeChan chan common.InterfaceMessage

	// GnbCpUe reads messages from all other workers and UE on this channel
	ReadChan chan common.InterfaceMessage

	// logger
	Log *logrus.Entry
}

func NewGnbCpUe(ngapId int64, gnb *GNodeB, amf *GnbAmf) *GnbCpUe {
	gnbue := GnbCpUe{}
	gnbue.GnbUeNgapId = ngapId
	gnbue.Amf = amf
	gnbue.Gnb = gnb
	gnbue.ReadChan = make(chan common.InterfaceMessage, 5)
	gnbue.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbCpUe",
		logger.FieldGnbUeNgapId: ngapId})
	gnbue.Log.Traceln("Context Created")
	return &gnbue
}

// GetGnbUpUe returns the GnbUpUe instance corresponding to provided PDU Sess ID
func (ctx *GnbCpUe) GetGnbUpUe(pduSessId int64) *GnbUpUe {
	ctx.Log.Infoln("Fetching GnbUpUe for pduSessId:", pduSessId)
	val, ok := ctx.GnbUpUes.Load(pduSessId)
	if ok {
		return val.(*GnbUpUe)
	} else {
		ctx.Log.Errorln("key not present:", pduSessId)
		return nil
	}
}

// AddGnbUpUe adds the GnbUpUe instance corresponding to provided PDU Sess ID
func (ctx *GnbCpUe) AddGnbUpUe(pduSessId int64, gnbue *GnbUpUe) {
	ctx.Log.Infoln("Adding new GnbUpUe for PDU Sess ID:", pduSessId)
	ctx.GnbUpUes.Store(pduSessId, gnbue)
}

// RemoveGnbUpUe removes the GnbUpUe instance corresponding to provided PDU
// sess ID from the map
func (ctx *GnbCpUe) RemoveGnbUpUe(pduSessId int64) {
	ctx.Log.Infoln("Deleting GnbUpUe for pduSessId:", pduSessId)
	ctx.GnbUpUes.Delete(pduSessId)
}
