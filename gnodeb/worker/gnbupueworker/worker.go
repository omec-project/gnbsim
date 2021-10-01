// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbcpueworker

import (
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"log"
)

func Init(gnbue *context.GnbUpUe) {
	for {
		msg := <-gnbue.ReadChan
		err := HandleMessage(gnbue, msg)
		if err != nil {
			log.Println(err)
		}
	}
}

func HandleMessage(gnbue *context.GnbUpUe, msg common.InterfaceMessage) (err error) {
	gnbue.Log.Infoln("Handling event:", msg.GetEventType(), "from interface:",
		msg.GetInterfaceType())
	switch msg.GetInterfaceType() {
	case common.UU_INTERFACE:
		uemsg := msg.(*common.UuMessage)
		switch uemsg.GetEventType() {

		}

	case common.N3_INTERFACE:
		upfmsg := msg.(*common.N2Message)
		switch upfmsg.GetEventType() {

		}
	}
	return nil
}

func SendToUe(gnbue *context.GnbCpUe, event common.EventType, nasPdus common.NasPduList) {
	gnbue.Log.Infoln("Sending event", event, "to SimUe")
	uemsg := common.UuMessage{}
	uemsg.Event = event
	uemsg.Interface = common.UU_INTERFACE
	//uemsg.NasPdus = nasPdus
	gnbue.WriteUeChan <- &uemsg
}
