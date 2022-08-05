// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package httprouter

import (
	"log"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/gnbsim/common"
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

func HTTPExecuteGUIProfile(c *gin.Context) {

	logger.HttpLog.Infoln("ExecuteGUIProfile API called")
	var prof profCtx.Profile

	if c.Request.Method == "GET" {
		t, err := template.ParseFiles("/gnbsim/bin/gnbsim.gtpl")
		if err != nil {
			logger.HttpLog.Errorln("template parse failed : ", err)
		}
		t.Execute(c.Writer, nil)
	} else {
		c.Request.ParseForm()
		prof.Init()
		prof.ProfileType = c.Request.PostForm.Get("profileType")
		prof.Name = c.Request.PostForm.Get("profileName")
		enable := c.Request.FormValue("enable")
		if enable == "false" {
			prof.Enable = false
		} else {
			prof.Enable = true
		}

		ueCount, err := strconv.ParseInt(c.Request.FormValue("ueCount")[0:], 10, 32)
		if err != nil {
			logger.HttpLog.Errorln("uint convert failed : ", err)
			prof.UeCount = 1
		} else {
			prof.UeCount = int(ueCount)
		}
		prof.GnbName = c.Request.PostForm.Get("gnbName")
		prof.StartImsi = c.Request.PostForm.Get("startImsi")
		prof.DefaultAs = c.Request.PostForm.Get("defaultAs")
		prof.Key = c.Request.PostForm.Get("key")
		prof.Opc = c.Request.PostForm.Get("opc")
		prof.SeqNum = c.Request.PostForm.Get("sequenceNumber")
		plmn := models.PlmnId{}
		plmn.Mcc = c.Request.PostForm.Get("mcc")
		plmn.Mnc = c.Request.PostForm.Get("mnc")
		prof.Plmn = &plmn
		fmt.Println("Profile values : ", prof)
		var responseChan = make(chan common.InterfaceMessage)
		go profile.ExecuteProfile(&prof, profCtx.SummaryChan, responseChan)
		for intfcMsg := range responseChan {
			fmt.Println("handling responseChan")
			if intfcMsg.GetEventType() == common.QUIT_EVENT {
				return
			}

			result := "PASS"
			// Waiting for execution summary from profile routine
			msg, ok := intfcMsg.(*common.SummaryMessage)
			if !ok {
				logger.HttpLog.Fatalln("Invalid Message Type")
			}

			fmt.Fprintf(c.Writer, "Profile Name : %v \n", msg.ProfileName)
			fmt.Fprintf(c.Writer, "Profile Type : %v \n", msg.ProfileType)
			fmt.Fprintf(c.Writer, "Ue's passed : %v \n", msg.UePassedCount)
			fmt.Fprintf(c.Writer, "Ue's failed : %v \n", msg.UeFailedCount)
			logger.HttpLog.Infoln("Profile Name:", msg.ProfileName, ", Profile Type:", msg.ProfileType)
			logger.HttpLog.Infoln("Ue's Passed:", msg.UePassedCount, ", Ue's Failed:", msg.UeFailedCount)

			if len(msg.ErrorList) != 0 {
				result = "FAIL"
				logger.HttpLog.Infoln("Profile Errors:")
				fmt.Fprintf(c.Writer, "Profile Errors : ")
				for _, err := range msg.ErrorList {
					logger.HttpLog.Errorln(err)
					fmt.Fprintf(c.Writer, err.Error())
				}
			}
			logger.HttpLog.Infoln("Profile Status:", result)
			fmt.Fprintf(c.Writer, "Profile Status : %v \n", result)
			return
		}
	}

	/*requestBody, err := c.GetRawData()
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
	go profile.ExecuteProfile(&prof, profCtx.SummaryChan)*/
}

func HTTPExecuteProfile(c *gin.Context) {

	logger.HttpLog.Infoln("ExecuteProfile API called")
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
	//go profile.ExecuteProfile(&prof, profCtx.SummaryChan, nil)
}
