// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/ngap/aper"
	"github.com/omec-project/ngap/ngapType"
)

func PrintAndGetCause(cause *ngapType.Cause) (present int, value aper.Enumerated) {
	present = cause.Present
	switch cause.Present {
	case ngapType.CausePresentRadioNetwork:
		logger.NgapLog.Infof("cause RadioNetwork[%d]", cause.RadioNetwork.Value)
		value = cause.RadioNetwork.Value
	case ngapType.CausePresentTransport:
		logger.NgapLog.Infof("cause Transport[%d]", cause.Transport.Value)
		value = cause.Transport.Value
	case ngapType.CausePresentProtocol:
		logger.NgapLog.Infof("cause Protocol[%d]", cause.Protocol.Value)
		value = cause.Protocol.Value
	case ngapType.CausePresentNas:
		logger.NgapLog.Infof("cause Nas[%d]", cause.Nas.Value)
		value = cause.Nas.Value
	case ngapType.CausePresentMisc:
		logger.NgapLog.Infof("cause Misc[%d]", cause.Misc.Value)
		value = cause.Misc.Value
	default:
		logger.NgapLog.Errorf("invalid Cause group[%d]", cause.Present)
	}
	return
}
