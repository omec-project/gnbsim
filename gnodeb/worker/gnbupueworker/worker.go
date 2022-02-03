// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbupueworker

import (
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"sync"
)

func Init(gnbue *context.GnbUpUe, wg *sync.WaitGroup) {
	HandleEvents(gnbue)
	wg.Done()
}

func HandleEvents(gnbue *context.GnbUpUe) {
	var err error
	for {
		select {
		/* Reading Up link packets from UE*/
		case msg := <-gnbue.ReadUlChan:
			err = HandleUlMessage(gnbue, msg)
			if err != nil {
				gnbue.Log.Errorln("HandleUlMessage() returned:", err)
			}

		/* Reading Down link packets from UPF worker*/
		case msg := <-gnbue.ReadDlChan:
			err = HandleDlMessage(gnbue, msg)
			if err != nil {
				gnbue.Log.Errorln("failed to handle downlink gtp-u message:", err)
			}

		/* Reading commands from GnbCpUe (Control plane context)*/
		case msg := <-gnbue.ReadCmdChan:
			evt := msg.GetEventType()
			gnbue.Log.Infoln("Handling:", common.GetEvtString(evt))
			switch evt {
			case common.QUIT_EVENT:
				HandleQuitEvent(gnbue, msg)
				return
			}
		}
		//TODO: Handle Errors
	}
}
