// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"

	"github.com/omec-project/gnbsim/logger"
	"github.com/sirupsen/logrus"
)

// TODO: Need to separate out the DAOs

// GnbUeDao acts as a Data Access Object that stores and provides access to all
// the GNodeB instances
type GnbUeDao struct {
	Log *logrus.Entry

	ngapIdGnbCpUeMap sync.Map
	dlTeidGnbUpUeMap sync.Map

	// TODO:
	// ulTeidGnbUpUeMap sync.Map
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
	dao.Log.Infoln("Fetching GnbCpUe for RANUENGAPID:", gnbUeNgapId)
	val, ok := dao.ngapIdGnbCpUeMap.Load(gnbUeNgapId)
	if ok {
		return val.(*GnbCpUe)
	} else {
		dao.Log.Warnln("key not present:", gnbUeNgapId)
		return nil
	}
}

// AddGnbCpUe adds the GnbCpUe instance corresponding to provided NGAP ID
func (dao *GnbUeDao) AddGnbCpUe(gnbUeNgapId int64, gnbue *GnbCpUe) {
	dao.Log.Infoln("Adding new GnbCpUe for RANUENGAPID:", gnbUeNgapId)
	dao.ngapIdGnbCpUeMap.Store(gnbUeNgapId, gnbue)
}

// GetGnbUpUe returns the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) GetGnbUpUe(teid uint32, downlink bool) *GnbUpUe {
	dao.Log.Traceln("Fetching GnbUpUe for TEID:", teid, "Downlink:", downlink)
	var val interface{}
	var ok bool
	if downlink {
		val, ok = dao.dlTeidGnbUpUeMap.Load(teid)
	}

	if ok {
		return val.(*GnbUpUe)
	} else {
		dao.Log.Warnln("key not present:", teid, "Downlink TEID :", downlink)
		return nil
	}
}

// AddGnbUpUe adds the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) AddGnbUpUe(teid uint32, downlink bool, gnbue *GnbUpUe) {
	dao.Log.Infoln("Adding new GnbUpUe for TEID:", teid, "Downlink:", downlink)
	if downlink {
		dao.dlTeidGnbUpUeMap.Store(teid, gnbue)
	}
}

// RemoveGnbUpUe removes the GnbUpUe instance corresponding to provided TEID
func (dao *GnbUeDao) RemoveGnbUpUe(teid uint32, downlink bool) {
	dao.Log.Infoln("Removing GnbUpUe for TEID:", teid, "Downlink:", downlink)
	if downlink {
		dao.dlTeidGnbUpUeMap.Delete(teid)
	}
}
