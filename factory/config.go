// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

/*
 * AMF Configuration Factory
 */

package factory

import (
	"fmt"
	"gnbsim/gnodeb/context"
	gnbctx "gnbsim/gnodeb/context"
	profctx "gnbsim/profile/context"
)

const (
	GNBSIM_EXPECTED_CONFIG_VERSION string = "1.0.0"
	GNBSIM_DEFAULT_CONFIG_PATH            = "/gnbsim/config/gnb.conf"
)

type Config struct {
	Info          *Info          `yaml:"info"`
	Configuration *Configuration `yaml:"configuration"`
	Logger        *Logger        `yaml: "logger"`
}

type Info struct {
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type Configuration struct {
	Gnbs     map[string]*gnbctx.GNodeB `yaml:"gnbs"`
	Profiles []*profctx.Profile        `yaml:"profiles"`
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

	if len(c.Configuration.Profiles) == 0 {
		return fmt.Errorf("no profile information available")
	}

	return nil
}

func (c *Configuration) GetGNodeB(name string) (*context.GNodeB, error) {
	var err error
	gnb, ok := c.Gnbs[name]
	if !ok {
		err = fmt.Errorf("no corresponding gNodeB found for:%v", name)
	}
	return gnb, err
}
