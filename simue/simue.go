// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package simue

import (
	"fmt"
	"gnbsim/gnodeb"
	intfc "gnbsim/interfacecommon"
	"gnbsim/realue"
	"gnbsim/simue/context"
)

func Init(simUe *context.SimUe) {
	go realue.Init(simUe.RealUe)
	err := ConnectToGnb(simUe)
	if err != nil {
		simUe.Log.Errorln(err)
	}
	// Start Sim UE event generation/processing logic
}

func ConnectToGnb(simUe *context.SimUe) (err error) {
	uemsg := intfc.UuMessage{}
	uemsg.Event = intfc.UE_CONNECTION_REQ
	uemsg.UeChan = simUe.ReadChan
	uemsg.Supi = simUe.Supi

	gNb := simUe.GnB
	simUe.WriteGnbUeChan = gnodeb.RequestConnection(gNb, &uemsg)
	if simUe.WriteGnbUeChan == nil {
		simUe.Log.Errorln("Received empty channel")
		err = fmt.Errorf("failed to connect to gnodeb")
		return err
	}

	simUe.Log.Infof("Connected to gNodeB, Name:%v, IP:%v, Port:%v", gNb.GnbName,
		gNb.GnbIp, gNb.GnbPort)
	return nil
}
