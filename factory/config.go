// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

/*
 * gNBSim Configuration Factory
 */

package factory

import (
	"fmt"
	"os"

	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	profctx "github.com/omec-project/gnbsim/profile/context"
)

const (
	GNBSIM_EXPECTED_CONFIG_VERSION = "1.0.0"
	GNBSIM_DEFAULT_CONFIG_PATH     = "/gnbsim/config/gnb.conf"
)

type Config struct {
	Info          *Info          `yaml:"info"`
	Configuration *Configuration `yaml:"configuration"`
	Logger        *Logger        `yaml:"logger"`
}

type Info struct {
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type Configuration struct {
	Gnbs                     map[string]*gnbctx.GNodeB   `yaml:"gnbs"`
	CustomProfiles           map[string]*profctx.Profile `yaml:"customProfiles"`
	Profiles                 []*profctx.Profile          `yaml:"profiles"`
	Server                   HttpServer                  `yaml:"httpServer"`
	GoProfile                ProfileServer               `yaml:"goProfile"`
	SingleInterface          bool                        `yaml:"singleInterface"`
	ExecInParallel           bool                        `yaml:"execInParallel"`
	RunConfigProfilesAtStart bool                        `yaml:"runConfigProfilesAtStart"`
}

type ProfileServer struct {
	Enable bool `yaml:"enable"`
	Port   int  `yaml:"port"`
}

type HttpServer struct {
	IpAddr string `yaml:"ipAddr"`
	Port   string `yaml:"port"`
	Enable bool   `yaml:"enable"`
}

type Logger struct {
	LogLevel string `yaml:"logLevel"`
}

func (c *Config) GetVersion() string {
	if c.Info != nil && c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}

func (c *Config) Validate() (err error) {
	if c.Info == nil {
		return fmt.Errorf("Info field missing")
	}

	if c.Configuration == nil {
		return fmt.Errorf("Configuration field missing")
	}

	if len(c.Configuration.Gnbs) == 0 {
		return fmt.Errorf("no gnbs configured")
	}
	if c.Configuration.GoProfile.Enable {
		if c.Configuration.GoProfile.Port == 0 {
			c.Configuration.GoProfile.Port = 5000
		}
	}

	if c.Configuration.Server.IpAddr == "POD_IP" {
		podIP := os.Getenv("POD_IP")
		if podIP == "" && c.Configuration.Server.Enable {
			// an empty address would make the listener bind to every interface
			return fmt.Errorf("httpServer.ipAddr is POD_IP but the POD_IP environment variable is not set")
		}
		c.Configuration.Server.IpAddr = podIP
	}

	if c.Configuration.SingleInterface {
		for _, gnb := range c.Configuration.Gnbs {
			if gnb.GnbN3Ip == "POD_IP" {
				gnb.GnbN3Ip = os.Getenv("POD_IP")
			}
		}
	}

	if len(c.Configuration.Profiles) == 0 {
		return fmt.Errorf("no profile information available")
	}

	if len(c.Configuration.CustomProfiles) != 0 {
		for _, v := range c.Configuration.CustomProfiles {
			if err := v.BuildIterations(); err != nil {
				return err
			}
		}

		for _, v := range c.Configuration.CustomProfiles {
			c.Configuration.Profiles = append(c.Configuration.Profiles, v)
		}
	}

	return nil
}

func (c *Configuration) GetGNodeB(name string) (*gnbctx.GNodeB, error) {
	var err error
	gnb, ok := c.Gnbs[name]
	if !ok {
		err = fmt.Errorf("no corresponding gNodeB found for:%v", name)
	}
	return gnb, err
}
