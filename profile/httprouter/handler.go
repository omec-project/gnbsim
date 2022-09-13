// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package httprouter

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/gnbsim/logger"
	profile "github.com/omec-project/gnbsim/profile"
	profCtx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/openapi"
	"github.com/omec-project/openapi/models"
)

func HTTPStepProfile(c *gin.Context) {
	logger.HttpLog.Infoln("HTTPStepProfile!")
	profName, exists := c.Params.Get("profile-name")
	if exists == false {
		logger.HttpLog.Printf("Received HTTPStepProfile, but profile-name not found ")
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	err := profCtx.SendStepEventProfile(profName)
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
	if exists == false {
		logger.HttpLog.Printf("Received HTTPAddNewCallsProfile, but profile-name not found ")
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	var number int32
	n, ok := c.GetQuery("number")
	if ok == false {
		number = 1
	} else {
		n, _ := strconv.Atoi(n)
		number = int32(n)
	}

	err := profCtx.SendAddNewCallsEventProfile(profName, number)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{})
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, gin.H{})
	}
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

	prof.Init()
	go profile.ExecuteProfile(&prof, profCtx.SummaryChan)
	c.JSON(http.StatusOK, gin.H{})
}
