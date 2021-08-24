// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package context

import (
	"log"
	"sync"
)

// GnbDao acts as a Data Access Object that stores and provides access to all
// the GNodeB instances
type GnbUeDao struct {
	gnbUeMap sync.Map
}

// GetGNodeB returns the GNodeB instance corresponding to provided name
func (dao *GnbUeDao) GetGnbUe(gnbUeNgapId int64) *GnbUe {
	val, ok := dao.gnbUeMap.Load(gnbUeNgapId)
	if ok {
		return val.(*GnbUe)
	} else {
		log.Println("key not present:", gnbUeNgapId)
		return nil
	}
}

// GetGNodeB returns the GNodeB instance corresponding to provided name
func (dao *GnbUeDao) AddGnbUe(gnbUeNgapId int64, gnbue *GnbUe) {
	dao.gnbUeMap.Store(gnbUeNgapId, gnbue)
}
