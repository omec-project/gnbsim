// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnodeb

import (
	"fmt"
	"net"
	"os"
)

// GnbDao acts as a Data Access Object that stores and provides access to all
// the GNodeB instances
type GnbDao struct {
	gnbMap map[string]*GNodeB
}

// GetGnbDao creates and returns a new GnbDao instance
func GetGnbDao() *GnbDao {
	gnbdao := GnbDao{
		gnbMap: make(map[string]*GNodeB),
	}
	return &gnbdao
}

// ParseGnbConfig creates GNodeB instances from the parsed YAML configuration
func (gnbdao *GnbDao) ParseGnbConfig() error {
	// TODO Should add logic to parse config file and load the gnbMap
	addrs, err := net.LookupHost("amf")
	if err != nil {
		fmt.Println("Failed to resolve amf")
		return err
	}

	gnb := GNodeB{
		GnbIp:   os.Getenv("POD_IP"),
		GnbPort: 9487,
		GnbName: "gnodeb1",
		GnbId:   []byte("\x00\x01\x02"),
		DefaultAmf: &GnbAmf{
			AmfIp:           addrs[0],
			AmfPort:         38412,
			ServedGuamiList: NewServedGUAMIList(),
			PlmnSupportList: NewPlmnSupportList(),
		},
		Tac: []byte("\x00\x00\x01"),
	}

	gnbdao.gnbMap["gnodeb1"] = &gnb
	return nil
}

// GetGNodeB returns the GNodeB instance corresponding to provided name
func (gnbdao *GnbDao) GetGNodeB(name string) *GNodeB {
	return gnbdao.gnbMap[name]
}

// InitializeAllGnbs initializes all the GNodeB instances present within the
// gnbMap
func (gnbdao *GnbDao) InitializeAllGnbs() error {
	for name, gnb := range gnbdao.gnbMap {
		err := gnb.Init()
		if err != nil {
			fmt.Println("Failed to initialize gNodeB: ", name, "error :", err)
			return err
		}
	}
	return nil
}
