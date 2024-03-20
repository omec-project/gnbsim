// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
// Copyright 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package httprouter

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/gnbsim/factory"
	"github.com/omec-project/gnbsim/logger"
	profile "github.com/omec-project/gnbsim/profile"
	profCtx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/openapi"
	"github.com/omec-project/openapi/models"
)

func HTTPStepProfile(c *gin.Context) {
	logger.HttpLog.Infoln("HTTPStepProfile!")
	profName, exists := c.Params.Get("profile-name")
	if !exists {
		logger.HttpLog.Printf("Received HTTPStepProfile, but profile-name not found ")
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	err := profile.SendStepEventProfile(profName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{})
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, gin.H{})
	}
}

func HTTPAddNewCallsProfile(c *gin.Context) {
	logger.HttpLog.Infoln("HTTPAddNewCallsProfile!")
	profName, exists := c.Params.Get("profile-name")
	if !exists {
		logger.HttpLog.Printf("Received HTTPAddNewCallsProfile, but profile-name not found ")
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	var number int32
	n, ok := c.GetQuery("number")
	if !ok {
		number = 1
	} else {
		n, err := strconv.Atoi(n)
		if err != nil {
			log.Println(err)
		}
		number = int32(n)
	}

	err := profile.SendAddNewCallsEventProfile(profName, number)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{})
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, gin.H{})
	}
}

func HTTPExecuteConfigProfile(c *gin.Context) {
	logger.HttpLog.Infoln("ExecuteConfigProfile API called")
	c.JSON(http.StatusOK, gin.H{"Status": "Request received. Test run in Progress"})

	go func() {
		var profileWaitGrp sync.WaitGroup
		config := factory.AppConfig
		// start profile and wait for it to finish (success or failure)
		// Keep running gnbsim as long as profiles are not finished
		for _, profileVal := range config.Configuration.Profiles {
			if !profileVal.Enable {
				continue
			}
			profileWaitGrp.Add(1)

			profile.InitProfile(profileVal, profCtx.SummaryChan)

			go func(profileCtx *profCtx.Profile) {
				defer profileWaitGrp.Done()
				profile.ExecuteProfile(profileCtx, profCtx.SummaryChan)
			}(profileVal)

			if !config.Configuration.ExecInParallel {
				profileWaitGrp.Wait()
			}
		}

		if config.Configuration.ExecInParallel {
			profileWaitGrp.Wait()
		}
	}()
}

func HTTPExecuteProfile(c *gin.Context) {
	logger.HttpLog.Infoln("EcecuteProfile API called")
	var prof profCtx.Profile

	requestBody, err := c.GetRawData()
	if err != nil {
		logger.HttpLog.Errorf("Get Request Body error: %+v", err)
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Deserialize(&prof, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.HttpLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}
	logger.HttpLog.Debugf("%#v", prof)

	err = prof.Init()
	if err != nil {
		logger.HttpLog.Infoln("failed to initiale profile", err)
	}
	go profile.ExecuteProfile(&prof, profCtx.SummaryChan)
	c.JSON(http.StatusOK, gin.H{})
}
