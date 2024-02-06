// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/omec-project/aper"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/ngap/ngapType"
)

func PrintAndGetCause(cause *ngapType.Cause) (present int, value aper.Enumerated) {
	present = cause.Present
	switch cause.Present {
	case ngapType.CausePresentRadioNetwork:
		logger.NgapLog.Infof("Cause RadioNetwork[%d]\n", cause.RadioNetwork.Value)
		value = cause.RadioNetwork.Value
	case ngapType.CausePresentTransport:
		logger.NgapLog.Infof("Cause Transport[%d]\n", cause.Transport.Value)
		value = cause.Transport.Value
	case ngapType.CausePresentProtocol:
		logger.NgapLog.Infof("Cause Protocol[%d]\n", cause.Protocol.Value)
		value = cause.Protocol.Value
	case ngapType.CausePresentNas:
		logger.NgapLog.Infof("Cause Nas[%d]\n", cause.Nas.Value)
		value = cause.Nas.Value
	case ngapType.CausePresentMisc:
		logger.NgapLog.Infof("Cause Misc[%d]\n", cause.Misc.Value)
		value = cause.Misc.Value
	default:
		logger.NgapLog.Errorf("Invalid Cause group[%d]\n", cause.Present)
	}
	return
}
