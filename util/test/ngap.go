// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package test

import (
	"gnbsim/logger"

	"github.com/free5gc/aper"
	"github.com/free5gc/ngap/ngapType"
)

func PrintAndGetCause(cause *ngapType.Cause) (present int, value aper.Enumerated) {
	present = cause.Present
	switch cause.Present {
	case ngapType.CausePresentRadioNetwork:
		logger.NgapLog.Infoln("Cause RadioNetwork[%d]\n", cause.RadioNetwork.Value)
		value = cause.RadioNetwork.Value
	case ngapType.CausePresentTransport:
		logger.NgapLog.Infoln("Cause Transport[%d]\n", cause.Transport.Value)
		value = cause.Transport.Value
	case ngapType.CausePresentProtocol:
		logger.NgapLog.Infoln("Cause Protocol[%d]\n", cause.Protocol.Value)
		value = cause.Protocol.Value
	case ngapType.CausePresentNas:
		logger.NgapLog.Infoln("Cause Nas[%d]\n", cause.Nas.Value)
		value = cause.Nas.Value
	case ngapType.CausePresentMisc:
		logger.NgapLog.Infoln("Cause Misc[%d]\n", cause.Misc.Value)
		value = cause.Misc.Value
	default:
		logger.NgapLog.Errorln("Invalid Cause group[%d]\n", cause.Present)
	}
	return
}
