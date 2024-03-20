// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"sync"

	"github.com/omec-project/gnbsim/logger"
	"github.com/sirupsen/logrus"
)

// GnbPeerDao acts as a Data Access Object that stores and provides access to all
// the GnbUpf and GnbAmf instances
type GnbPeerDao struct {
	Log *logrus.Entry

	// Map of UPF IP address vs GnbUpf Context. Not considering Port as they
	// will be same for all UPFs i.e GTP-U port 2152
	gnbUpfMap sync.Map
	lock      sync.Mutex

	// TODO
	// gnbAmfMap sync.Map
}

func NewGnbPeerDao() *GnbPeerDao {
	dao := &GnbPeerDao{}
	dao.Log = logger.GNodeBLog.WithFields(logrus.Fields{"subcategory": "GnbPeerDao"})
	return dao
}

// GetGnbUpf returns the GnbUpf instance corresponding to provided IP
func (dao *GnbPeerDao) GetGnbUpf(ip string) *GnbUpf {
	dao.Log.Infoln("Fetching GnbUpf corresponding to IP:", ip)
	val, ok := dao.gnbUpfMap.Load(ip)
	if ok {
		return val.(*GnbUpf)
	} else {
		dao.Log.Warnln("key not present:", ip)
		return nil
	}
}

func (dao *GnbPeerDao) GetOrAddGnbUpf(ip string) (*GnbUpf, bool) {
	// Though it is a sync map, we need to acquire lock because this function
	// can be called from multiple Go routines, in which case the fetch + add
	// operation should be atomic
	dao.lock.Lock()
	defer dao.lock.Unlock()

	var created bool
	gnbupf := dao.GetGnbUpf(ip)
	if gnbupf == nil {
		created = true
		gnbupf = NewGnbUpf(ip)
		dao.AddGnbUpf(ip, gnbupf)
	}
	return gnbupf, created
}

// AddGnbUpf adds a GnbUpf instance corresponding to the IP into the map
func (dao *GnbPeerDao) AddGnbUpf(ip string, gnbupf *GnbUpf) {
	dao.Log.Infoln("Adding new GnbUpf corresponding to IP:", ip)
	dao.gnbUpfMap.Store(ip, gnbupf)
}
