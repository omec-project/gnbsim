// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package pdusessworker

import (
	"gnbsim/common"
	"gnbsim/realue/context"
)

func Init(pduSess *context.PduSession) {
	for {
		select {
		/* Reading Down link packets from gNb*/
		case msg := <-pduSess.ReadDlChan:
			err := HandleDlMessage(pduSess, msg)
			if err != nil {
				pduSess.Log.Errorln("HandleDlMessage() returned:", err)
			}

		/* Reading commands from RealUE control plane*/
		case msg := <-pduSess.ReadCmdChan:
			err := HandleCommand(pduSess, msg)
			if err != nil {
				pduSess.Log.Errorln("HandleCommand() returned:", err)
			}
		}
	}
}

func HandleCommand(gnbue *context.PduSession, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Infoln("Handling event:", msg.GetEventType())
	uemsg := msg.(*common.UuMessage)
	switch uemsg.GetEventType() {

	}

	return nil
}
