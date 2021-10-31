// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package main

import (
	"gnbsim/factory"
	"gnbsim/gnodeb"
	"gnbsim/logger"
	"gnbsim/profile"
	"gnbsim/profile/deregister"
	"gnbsim/profile/ngsetup"
	"gnbsim/profile/pdusessest"
	"gnbsim/profile/register"
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "GNBSIM"
	app.Usage = "./gnbsim -cfg [gnbsim configuration file]"
	app.Action = action
	app.Flags = getCliFlags()

	logger.AppLog.Infoln("App Name:", app.Name)

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Errorln("Failed to run GNBSIM:", err)
		return
	}
}

func action(c *cli.Context) error {
	cfg := c.String("cfg")
	if cfg == "" {
		logger.AppLog.Warnln("No configuration file provided. Using default configuration file:", factory.GNBSIM_DEFAULT_CONFIG_PATH)
		logger.AppLog.Infoln("Application Usage:", c.App.Usage)
		cfg = factory.GNBSIM_DEFAULT_CONFIG_PATH
	}

	if err := factory.InitConfigFactory(cfg); err != nil {
		logger.AppLog.Errorln("Failed to initialize config factory:", err)
		return err
	}

	config := factory.AppConfig
	lvl := config.Logger.LogLevel
	logger.AppLog.Infoln("Setting log level to:", lvl)
	logger.SetLogLevel(lvl)

	profile.InitializeAllProfiles()
	err := gnodeb.InitializeAllGnbs()
	if err != nil {
		logger.AppLog.Errorln("Failed to initialize gNodeBs:", err)
		return err
	}

	for _, profileCtx := range config.Configuration.Profiles {
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
			case "pdusessest":
				{
					pdusessest.PduSessEst_test(profileCtx)
				}
			case "deregister":
				{
					deregister.Deregister_test(profileCtx)
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
					//loadsub.LoadSubscriberData(10)
				}
			}
		}
	}
	return nil
}

func getCliFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "cfg",
			Usage: "GNBSIM config file",
		},
	}
}
