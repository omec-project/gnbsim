// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbupueworker

import (
	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
)

func Init(gnbue *gnbctx.GnbUpUe) {
	HandleEvents(gnbue)
}

func HandleEvents(gnbue *gnbctx.GnbUpUe) {
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
			gnbue.Log.Infoln("Handling:", evt)
			switch evt {
			case common.QUIT_EVENT:
				err := HandleQuitEvent(gnbue, msg)
				if err != nil {
					gnbue.Log.Errorln("failed to handle quiet event:", err)
				}
				return
			}
		}
		// TODO: Handle Errors
	}
}
