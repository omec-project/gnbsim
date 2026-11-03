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
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/utils"
)

const (
	// maxRequestBodyBytes bounds the profile JSON accepted by executeProfile.
	maxRequestBodyBytes = 1 << 20
	// maxAddNewCalls bounds the "number" query parameter of addNewCalls.
	maxAddNewCalls = 10000
)

func HTTPStepProfile(c *gin.Context) {
	logger.HttpLog.Infoln("HTTPStepProfile")
	profName, exists := c.Params.Get("profile-name")
	if !exists {
		logger.HttpLog.Warnln("received HTTPStepProfile but profile-name not found")
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
		logger.HttpLog.Warnln("received HTTPAddNewCallsProfile, but profile-name not found")
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	var number int32
	n, ok := c.GetQuery("number")
	if !ok {
		number = 1
	} else {
		n, err := strconv.Atoi(n)
		if err != nil || n < 1 || n > maxAddNewCalls {
			logger.HttpLog.Warnf("invalid number of calls %q, expected an integer in [1, %d]", c.Query("number"), maxAddNewCalls)
			c.JSON(http.StatusBadRequest, gin.H{})
			return
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

			if err := profile.InitProfile(profileVal, profCtx.SummaryChan); err != nil {
				logger.HttpLog.Errorf("failed to initialize profile %s: %v", profileVal.Name, err)
				profileWaitGrp.Done()
				continue
			}

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
	logger.HttpLog.Infoln("ExecuteProfile API called")
	var prof profCtx.Profile

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
	requestBody, err := c.GetRawData()
	if err != nil {
		logger.HttpLog.Errorf("get Request Body error: %+v", err)
		problemDetail := utils.ProblemDetailsSystemFailure(err.Error())
		c.JSON(http.StatusInternalServerError, problemDetail)
		return
	}

	err = openapi.Decode(&prof, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := utils.ProblemDetailsMalformedRequestSyntax(problemDetail)
		logger.HttpLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}
	// the profile carries the subscriber key and OPc, so it is deliberately not logged

	err = prof.Init()
	if err != nil {
		logger.HttpLog.Errorln("failed to initialize profile:", err)
		c.JSON(http.StatusBadRequest, utils.ProblemDetailsMalformedRequestSyntax("[Profile] "+err.Error()))
		return
	}
	// create the SimUe contexts; ExecuteProfile has nothing to run without them
	if err = profile.InitProfile(&prof, profCtx.SummaryChan); err != nil {
		logger.HttpLog.Errorln("failed to create profile UEs:", err)
		c.JSON(http.StatusBadRequest, utils.ProblemDetailsMalformedRequestSyntax("[Profile] "+err.Error()))
		return
	}
	go profile.ExecuteProfile(&prof, profCtx.SummaryChan)
	c.JSON(http.StatusOK, gin.H{})
}
