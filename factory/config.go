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
	"strconv"
	"strings"

	"github.com/omec-project/gnbsim/common"
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
			it := v.Iterations
			v.PIterations = make(map[string]*profctx.PIterations)
			for _, v1 := range it {
				if len(v1.Next) == 0 {
					v1.Next = "quit" // default value
				}
				PIter := &profctx.PIterations{Name: v1.Name, NextItr: v1.Next, Repeat: v1.Repeat}
				PIter.ProcMap = make(map[int]common.ProcedureType)
				PIter.WaitMap = make(map[int]int)
				PIter.WaitMap[0] = 0
				steps := []string{v1.First, v1.Second, v1.Third, v1.Fourth, v1.Fifth, v1.Sixth, v1.Seventh}
				for i, step := range steps {
					if len(step) == 0 {
						continue
					}
					proc, wait, perr := parseIterationStep(step)
					if perr != nil {
						return fmt.Errorf("iteration %q step %d: %v", v1.Name, i+1, perr)
					}
					PIter.ProcMap[i+1] = proc
					PIter.WaitMap[i+1] = wait
				}
				v.PIterations[v1.Name] = PIter // add iterations in the custom profile
			}
			// every reference must resolve, otherwise the profile dereferences a nil iteration at run time
			if len(v.PIterations) > 0 {
				if _, ok := v.PIterations[v.StartIteration]; !ok {
					return fmt.Errorf("custom profile %q: startiteration %q is not a defined iteration", v.Name, v.StartIteration)
				}
			}
			for _, pit := range v.PIterations {
				if pit.NextItr == "quit" {
					continue
				}
				if _, ok := v.PIterations[pit.NextItr]; !ok {
					return fmt.Errorf("custom profile %q: iteration %q has unknown next iteration %q", v.Name, pit.Name, pit.NextItr)
				}
			}
		}

		for _, v := range c.Configuration.CustomProfiles {
			c.Configuration.Profiles = append(c.Configuration.Profiles, v)
		}
	}

	return nil
}

// parseIterationStep parses a custom profile step of the form "<PROCEDURE-NAME> <wait-ms>".
func parseIterationStep(step string) (common.ProcedureType, int, error) {
	fields := strings.Fields(step)
	if len(fields) != 2 {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("expected \"<procedure> <wait>\", got %q", step)
	}
	proc := common.GetProcId(fields[0])
	if proc == common.UNKNOWN_PROCEDURE {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("unknown procedure %q", fields[0])
	}
	wait, err := strconv.Atoi(fields[1])
	if err != nil {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("value is not converted to integer: %v", err)
	}
	if wait < 0 {
		return common.UNKNOWN_PROCEDURE, 0, fmt.Errorf("wait must not be negative, got %d", wait)
	}
	return proc, wait, nil
}

func (c *Configuration) GetGNodeB(name string) (*gnbctx.GNodeB, error) {
	var err error
	gnb, ok := c.Gnbs[name]
	if !ok {
		err = fmt.Errorf("no corresponding gNodeB found for:%v", name)
	}
	return gnb, err
}
