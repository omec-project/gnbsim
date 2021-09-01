// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package main

import (
	"fmt"
	"gnbsim/gnodeb/dao"
	"gnbsim/loadsub"
	"gnbsim/profile/context"
	"gnbsim/profile/ngsetup"
	"gnbsim/profile/register"
	"os"
)

func main() {
	fmt.Println("Main function")
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "(ngsetup | loadsubs | register | deregister | xnhandover | paging | n2handover | servicereq | servicereqmacfail | resynchronisation | gutiregistration | duplicateregistration | pdusessionrelease)")
		return
	}
	testcase := os.Args[1]

	fmt.Println("argsWithoutProg ", testcase)

	ranIpAddr := os.Getenv("POD_IP")
	fmt.Println("Hello World from RAN - ", ranIpAddr)

	gnbDao := dao.GetGnbDao()
	err := gnbDao.ParseGnbConfig()
	if err != nil {
		fmt.Println("Failed to parse config")
		return
	}
	err = gnbDao.InitializeAllGnbs()
	if err != nil {
		fmt.Println("Failed to initialize gNodeBs")
		return
	}

	switch testcase {
	case "ngsetup":
		{
			fmt.Println("test ngsetup")
			// TODO which gnb to use should be parsed from the config file
			gnb := gnbDao.GetGNodeB("gnodeb1")
			ngsetup.NgSetup_test(gnb)
		}
	case "register":
		{
			fmt.Println("test register")
			// TODO parse profile from config file
			profileCtx := context.NewProfile("register")
			gnb := gnbDao.GetGNodeB(profileCtx.GnbName)
			register.Register_test(profileCtx, gnb)
		}
	case "deregister":
		{
			fmt.Println("test deregister")
			//deregister.Deregister_test(ranIpAddr, amfIpAddr)
		}
	case "pdusessionrelease":
		{
			fmt.Println("test pdusessionrelease")
			//pdusessionrelease.PduSessionRelease_test(ranIpAddr, amfIpAddr)
		}
	case "duplicateregistration":
		{
			fmt.Println("test duplicateregistration")
			//duplicateregistration.DuplicateRegistration_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "gutiregistration":
		{
			fmt.Println("test gutiregistration")
			//gutiregistration.Gutiregistration_test(ranIpAddr, amfIpAddr)
		}
	case "n2handover":
		{
			fmt.Println("test n2handover")
			//n2handover.N2Handover_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "paging":
		{
			fmt.Println("test paging")
			//paging.Paging_test(ranIpAddr, amfIpAddr)
		}
	case "resynchronisation":
		{
			fmt.Println("test resynchronisation")
			//resynchronisation.Resychronisation_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "servicereqmacfail":
		{
			fmt.Println("test servicereq macfail")
			//servicereq.Servicereq_macfail_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "servicereq":
		{
			fmt.Println("test servicereq")
			//servicereq.Servicereq_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "xnhandover":
		{
			fmt.Println("test xnhandover")
			//xnhandover.Xnhandover_test(ranUIpAddr, ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "loadsubs":
		{
			fmt.Println("loading subscribers in DB")
			loadsub.LoadSubscriberData(10)
		}
	}

	return
}
