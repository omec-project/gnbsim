// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package ngsetup

import (
	"fmt"
	"gnbsim/gnodeb"
	"net"
)

func NgSetup_test(gnb *gnodeb.GNodeB) {
	// create amf
	addrs, err := net.LookupHost("amf")
	if err != nil {
		fmt.Println("Failed to resolve amf")
		return
	}
	gnbamf := gnodeb.NewGnbAmf(addrs[0], 38412)

	err = gnb.ConnectToAmf(gnbamf)
	if err != nil {
		fmt.Println("ConnectToAmf() failed due to:", err)
		return
	}

	successFulOutcome, err := gnb.PerformNgSetup(gnbamf)
	if err != nil {
		fmt.Println("PerformNGSetup() failed due to:", err)
	} else if !successFulOutcome {
		fmt.Println("Expected SuccessfulOutcome, received UnsuccessfulOutcome")
		return
	}

	fmt.Println("NGSetup Procedure successful")
}
