// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package factory

import "testing"

func TestInitConfigFactoryLoadsGlobalRanID(t *testing.T) {
	if err := InitConfigFactory("../config/gnbsim.yaml"); err != nil {
		t.Fatalf("InitConfigFactory returned error: %v", err)
	}

	gnb, ok := AppConfig.Configuration.Gnbs["gnb1"]
	if !ok {
		t.Fatal("expected gnb1 to be present in configuration")
	}

	if gnb.RanId.GetPlmnId().Mcc != "208" || gnb.RanId.GetPlmnId().Mnc != "93" {
		t.Fatalf("unexpected PLMN in globalRanId: %+v", gnb.RanId.GetPlmnId())
	}

	if gnb.RanId.GNbId == nil {
		t.Fatal("expected globalRanId.gNbId to be populated")
	}

	if gnb.RanId.GetGNbId().BitLength != 24 || gnb.RanId.GetGNbId().GNBValue != "001001" {
		t.Fatalf("unexpected gNbId in globalRanId: %+v", gnb.RanId.GetGNbId())
	}
}
