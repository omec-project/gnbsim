// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbupfworker

import (
	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/gnbsim/util/test"
)

/* HandleNGSetupResponse processes the NG Setup Response and updates GnbAmf
 * context
 */
func HandleDlGpduMessage(gnbUpf *gnbctx.GnbUpf, gtpPdu *test.GtpPdu) error {
	gnbUpf.Log.Traceln("Processing downlink G-PDU packet")
	gnbUpUe := gnbUpf.GnbUpUes.GetGnbUpUe(gtpPdu.Hdr.Teid, true)
	if gnbUpUe == nil {
		return nil
		/* TODO: Send ErrorIndication message to upf*/
	}
	msg := &common.N3Message{}
	msg.Event = common.DL_UE_DATA_TRANSPORT_EVENT
	msg.Pdu = gtpPdu
	gnbUpUe.ReadDlChan <- msg

	return nil
}
