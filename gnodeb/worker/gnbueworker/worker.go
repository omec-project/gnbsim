// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbueworker

import (
	"gnbsim/gnodeb/context"
	intfc "gnbsim/interfacecommon"
	"log"
)

func Init(gnbue *context.GnbUe) {
	for {
		select {
		case msg := <-gnbue.ReadChan:
			err := HandleMessage(gnbue, msg)
			if err != nil {
				log.Println(err)
			}
		case <-gnbue.Gnb.Quit:
			return
		}
	}
}

func HandleMessage(gnbue *context.GnbUe, msg intfc.InterfaceMessage) (err error) {
	log.Println("Recived First Message from UE, YIPPIE")
	switch msg.GetInterfaceType() {
	case intfc.UU_INTERFACE:
		uemsg := msg.(*intfc.UuMessage)
		switch uemsg.GetEventType() {
		case intfc.UE_CONNECTION_REQ:
			HandleUeConnection(gnbue, uemsg)
		case intfc.UE_REG_REQUEST:
			HandleInitialUEMessage(gnbue, uemsg)
		case intfc.UE_UPLINK_NAS_TRANSPORT:
			HandleUplinkNasTransport(gnbue, uemsg)
		}

	case intfc.N2_INTERFACE:
		amfmsg := msg.(*intfc.N2Message)
		switch msg.GetEventType() {
		case intfc.AMF_DOWNLINK_NAS_TRANSPORT:
			HandleDownlinkNasTransport(gnbue, amfmsg)
		case intfc.AMF_INITIAL_CONTEXT_SETUP_REQUEST:
			HandleInitialContextSetupRequest(gnbue, amfmsg)
		}
	}
	return nil
}
