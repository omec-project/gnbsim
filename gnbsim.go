// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package main

import (
	"fmt"
	"gnbsim/factory"
	"gnbsim/gnodeb"
	"gnbsim/loadsub"
	"gnbsim/logger"
	"gnbsim/profile"
	"gnbsim/profile/ngsetup"
	"gnbsim/profile/register"
)

func main() {
	if err := factory.InitConfigFactory(factory.GNBSIM_DEFAULT_CONFIG_PATH); err != nil {
		logger.AppLog.Errorln("Failed to initialize config factory")
		return
	}

	config := factory.AppConfig.Configuration

	profile.InitializeAllProfiles()
	err := gnodeb.InitializeAllGnbs()
	if err != nil {
		fmt.Println("Failed to initialize gNodeBs")
		return
	}

	for _, profileCtx := range config.Profiles {
		if profileCtx.Enable {
			logger.AppLog.Infoln("executing profile:", profileCtx.Name,
				", profile type:", profileCtx.ProfileType)

			switch profileCtx.ProfileType {
			case "ngsetup":
				{
					ngsetup.NgSetup_test(profileCtx)
				}
			case "register":
				{
					register.Register_test(profileCtx)
				}
			case "deregister":
				{
					//deregister.Deregister_test(ranIpAddr, amfIpAddr)
				}
			case "pdusessionrelease":
				{
					//pdusessionrelease.PduSessionRelease_test(ranIpAddr, amfIpAddr)
				}
			case "duplicateregistration":
				{
					//duplicateregistration.DuplicateRegistration_test(ranIpAddr, upfIpAddr, amfIpAddr)
				}
			case "gutiregistration":
				{
					//gutiregistration.Gutiregistration_test(ranIpAddr, amfIpAddr)
				}
			case "n2handover":
				{
					//n2handover.N2Handover_test(ranIpAddr, upfIpAddr, amfIpAddr)
				}
			case "paging":
				{
					//paging.Paging_test(ranIpAddr, amfIpAddr)
				}
			case "resynchronisation":
				{
					//resynchronisation.Resychronisation_test(ranIpAddr, upfIpAddr, amfIpAddr)
				}
			case "servicereqmacfail":
				{
					//servicereq.Servicereq_macfail_test(ranIpAddr, upfIpAddr, amfIpAddr)
				}
			case "servicereq":
				{
					//servicereq.Servicereq_test(ranIpAddr, upfIpAddr, amfIpAddr)
				}
			case "xnhandover":
				{
					//xnhandover.Xnhandover_test(ranUIpAddr, ranIpAddr, upfIpAddr, amfIpAddr)
				}
			case "loadsubs":
				{
					loadsub.LoadSubscriberData(10)
				}
			}
		}
	}
}
