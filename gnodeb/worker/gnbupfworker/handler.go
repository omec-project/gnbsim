// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbupfworker

import (
	"gnbsim/common"
	"gnbsim/gnodeb/context"
	"gnbsim/util/test"
)

/* HandleNGSetupResponse processes the NG Setup Response and updates GnbAmf
 * context
 */
func HandleDlGpduMessage(gnbUpf *context.GnbUpf, gtpHdr *test.GtpHdr,
	optHdr *test.GtpHdrOpt, payload []byte) error {
	gnbUpUe := gnbUpf.GnbUpUes.GetGnbUpUe(gtpHdr.Teid, true)
	if gnbUpUe == nil {
		return nil
		/* TODO: Send ErrorIndication message to upf*/
	}
	userDataMsg := &common.UserDataMessage{}
	userDataMsg.Event = common.DL_UE_DATA_TRANSFER_EVENT
	userDataMsg.Interface = common.N3_INTERFACE
	userDataMsg.Payload = payload
	gnbUpUe.ReadDlChan <- userDataMsg

	return nil
}
