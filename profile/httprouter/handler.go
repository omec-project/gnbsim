// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package httprouter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/gnbsim/logger"
	profile "github.com/omec-project/gnbsim/profile"
	profCtx "github.com/omec-project/gnbsim/profile/context"
	"github.com/omec-project/openapi"
	"github.com/omec-project/openapi/models"
)

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
}
