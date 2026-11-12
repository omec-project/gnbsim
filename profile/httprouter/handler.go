// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
// Copyright 2022 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package httprouter

import (
	"encoding/hex"
	"errors"
	"fmt"
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
	// maxUeCount bounds the ueCount of a profile received over REST.
	maxUeCount = 10000
	// maxDataPktCount, maxDataPktInterval (seconds) and maxPerUserTimeout (seconds) bound what a
	// REST profile can ask of the packet generator and of a UE's time.
	maxDataPktCount    = 100000
	maxDataPktInterval = 3600
	maxPerUserTimeout  = 86400
	// subscriberKeyHexLen is the length of K and OPc (128 bits) in hexadecimal characters.
	subscriberKeyHexLen = 32
)

// validateRestProfile rejects profile values that would otherwise terminate the process once the
// UEs start (nil PLMN, non-hexadecimal K or OPc) or exhaust memory (unbounded ueCount). Profiles
// from the configuration file are trusted and not checked here.
func validateRestProfile(prof *profCtx.Profile) error {
	if prof.UeCount < 1 || prof.UeCount > maxUeCount {
		return fmt.Errorf("ueCount must be an integer in [1, %d]", maxUeCount)
	}
	if prof.DataPktCount > maxDataPktCount || prof.DataPktInt > maxDataPktInterval {
		return fmt.Errorf("dataPktCount must be at most %d and dataPktInterval at most %d", maxDataPktCount, maxDataPktInterval)
	}
	if prof.PerUserTimeout > maxPerUserTimeout {
		return fmt.Errorf("perUserTimeout must be at most %d", maxPerUserTimeout)
	}
	if prof.Plmn == nil {
		return errors.New("plmnId is required")
	}
	if !isDigits(prof.Plmn.Mcc, 3, 3) {
		return errors.New("plmnId.mcc must be 3 digits")
	}
	if !isDigits(prof.Plmn.Mnc, 2, 3) {
		return errors.New("plmnId.mnc must be 2 or 3 digits")
	}
	for name, v := range map[string]string{"key": prof.Key, "opc": prof.Opc} {
		if b, err := hex.DecodeString(v); err != nil || len(b)*2 != subscriberKeyHexLen {
			return fmt.Errorf("%s must be %d hexadecimal characters", name, subscriberKeyHexLen)
		}
	}
	return nil
}

func isDigits(s string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

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
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge,
				utils.ProblemDetailsMalformedRequestSyntax(fmt.Sprintf("request body exceeds %d bytes", maxRequestBodyBytes)))
			return
		}
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

	if err = validateRestProfile(&prof); err != nil {
		logger.HttpLog.Errorln("invalid profile:", err)
		c.JSON(http.StatusBadRequest, utils.ProblemDetailsMalformedRequestSyntax("[Profile] "+err.Error()))
		return
	}

	err = prof.Init()
	if err != nil {
		logger.HttpLog.Errorln("failed to initialize profile:", err)
		c.JSON(http.StatusBadRequest, utils.ProblemDetailsMalformedRequestSyntax("[Profile] "+err.Error()))
		return
	}
	// create the SimUe contexts; ExecuteProfile has nothing to run without them. This also
	// publishes the profile, so step and add-calls requests only see it once it is complete.
	if err = profile.InitProfile(&prof, profCtx.SummaryChan); err != nil {
		logger.HttpLog.Errorln("failed to create profile UEs:", err)
		c.JSON(http.StatusBadRequest, utils.ProblemDetailsMalformedRequestSyntax("[Profile] "+err.Error()))
		return
	}
	go profile.ExecuteProfile(&prof, profCtx.SummaryChan)
	c.JSON(http.StatusOK, gin.H{})
}
