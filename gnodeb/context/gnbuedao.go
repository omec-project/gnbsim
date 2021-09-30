// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"gnbsim/logger"
	"log"
	"sync"

	"github.com/sirupsen/logrus"
)

// GnbUeDao acts as a Data Access Object that stores and provides access to all
// the GNodeB instances
type GnbUeDao struct {
	ngapIdGnbCpUeMap sync.Map
	dlTeidGnbUpUeMap sync.Map

	/* logger */
	Log *logrus.Entry
	//TODO:
	//ulTeidGnbUpUeMao sync.Map
	// This map will be helpful when gNb receives an ErrorIndication Message
	// which will have an UL TEID. In which case gNb can fetch and delete the
	// GnbUpUe context corresponding to that UL TEID
}

func NewGnbUeDao() *GnbUeDao {
	dao := &GnbUeDao{}
	dao.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbUeDao"})
	return dao
}

// GetGnbCpUe returns the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbUeDao) GetGnbCpUe(gnbUeNgapId int64) *GnbCpUe {
	val, ok := dao.ngapIdGnbCpUeMap.Load(gnbUeNgapId)
	if ok {
		return val.(*GnbCpUe)
	} else {
		log.Println("key not present:", gnbUeNgapId)
		return nil
	}
}

// AddGnbCpUe adds the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbUeDao) AddGnbCpUe(gnbUeNgapId int64, gnbue *GnbCpUe) {
	dao.ngapIdGnbCpUeMap.Store(gnbUeNgapId, gnbue)
}

// GetGnbCpUe returns the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbUeDao) GetGnbUpUe(teid int64, downlink bool) *GnbUpUe {
	var val interface{}
	var ok bool
	if downlink {
		val, ok = dao.dlTeidGnbUpUeMap.Load(teid)
	} else {
		// TODO
		//val, ok = dao.ulTeidGnbUpUeMap.Load(teid)
	}

	if ok {
		return val.(*GnbUpUe)
	} else {
		log.Println("key not present:", teid, "Downlink TEID :", downlink)
		return nil
	}
}

// AddGnbCpUe adds the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbCpUeDao) AddGnbUpUe(gnbUeNgapId int64, gnbue *GnbCpUe) {
	dao.ngapIdGnbCpUeMap.Store(gnbUeNgapId, gnbue)
}
