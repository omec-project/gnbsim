// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package pdusessworker

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/realue/context"
	"sync"
)

func Init(pduSess *context.PduSession, wg *sync.WaitGroup) {
	HandleEvents(pduSess)
	wg.Done()
}

func HandleEvents(pduSess *context.PduSession) {
	var err error
	for {
		select {
		/* Reading Down link packets from gNb*/
		case msg := <-pduSess.ReadDlChan:
			err = HandleDlMessage(pduSess, msg)
		/* Reading commands from RealUE control plane*/
		case msg := <-pduSess.ReadCmdChan:
			event := msg.GetEventType()
			pduSess.Log.Infoln("Handling:", common.GetEvtString(event))

			switch event {
			case common.INIT_EVENT:
				HandleInitEvent(pduSess, msg)
			case common.DATA_PKT_GEN_REQUEST_EVENT:
				err = HandleDataPktGenRequestEvent(pduSess, msg)
			case common.CONNECTION_RELEASE_REQUEST_EVENT:
				err = HandleConnectionReleaseRequestEvent(pduSess, msg)
			case common.QUIT_EVENT:
				HandleQuitEvent(pduSess, msg)
				return
			}
		}

		if err != nil {
			msg := &common.UeMessage{}
			msg.Error = fmt.Errorf("pdu session failed:%v", err)
			msg.Event = common.ERROR_EVENT
			pduSess.WriteUeChan <- msg
		}
	}
}
