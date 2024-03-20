// SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0

package httprouter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/gnbsim/logger"
	"github.com/omec-project/logger_util"
)

// Route is the information for every URI.
type Route struct {
	// HandlerFunc is the handler function of this route.
	HandlerFunc gin.HandlerFunc
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
}

// Routes is the list of the generated Route.
type Routes []Route

// NewRouter returns a new router.
func NewRouter() *gin.Engine {
	router := logger_util.NewGinWithLogrus(logger.GinLog)
	AddService(router)
	return router
}

func AddService(engine *gin.Engine) *gin.RouterGroup {
	group := engine.Group("/gnbsim/v1")

	for _, route := range routes {
		switch route.Method {
		case "GET":
			group.GET(route.Pattern, route.HandlerFunc)
		case "POST":
			group.POST(route.Pattern, route.HandlerFunc)
		case "PUT":
			group.PUT(route.Pattern, route.HandlerFunc)
		case "PATCH":
			group.PATCH(route.Pattern, route.HandlerFunc)
		case "DELETE":
			group.DELETE(route.Pattern, route.HandlerFunc)
		}
	}
	return group
}

// Index is the index handler.
func Index(c *gin.Context) {
	c.String(http.StatusOK, "Hello World!")
}

var routes = Routes{
	{
		Name: "Index", Method: "GET",
		Pattern: "/", HandlerFunc: Index,
	},

	{
		Name: "ExecuteProfile", Method: "POST",
		Pattern: "/executeProfile", HandlerFunc: HTTPExecuteProfile,
	},
	{
		Name: "ExecuteConfigProfile", Method: "POST",
		Pattern: "/executeConfigProfile", HandlerFunc: HTTPExecuteConfigProfile,
	},
	{
		Name: "StepProfile", Method: "POST",
		Pattern: "/:profile-name/stepProfile", HandlerFunc: HTTPStepProfile,
	},
	{
		Name: "addNewCalls", Method: "POST",
		Pattern: "/:profile-name/addNewCalls", HandlerFunc: HTTPAddNewCallsProfile,
	},
}
