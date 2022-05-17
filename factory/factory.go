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
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/omec-project/amf/logger"
)

var AppConfig *Config

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	content, err := ioutil.ReadFile(f)
	if err != nil {
		logger.CfgLog.Errorln("Failed to read", f, "file:", err)
		return err
	}

	AppConfig = &Config{}

	err = yaml.Unmarshal(content, AppConfig)
	if err != nil {
		logger.CfgLog.Errorln("Failed to unmarshal:", err)
		return err
	}

	err = AppConfig.Validate()
	if err != nil {
		logger.CfgLog.Errorln("Invalid Configuration:", err)
	}

	return err
}

func CheckConfigVersion() error {
	currentVersion := AppConfig.GetVersion()

	if currentVersion != GNBSIM_EXPECTED_CONFIG_VERSION {
		return fmt.Errorf("config version is [%s], but expected is [%s].",
			currentVersion, GNBSIM_EXPECTED_CONFIG_VERSION)
	}

	logger.CfgLog.Infof("config version [%s]", currentVersion)

	return nil
}
