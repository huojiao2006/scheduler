// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package apiserver

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"

	"openpitrix.io/newbilling/pkg/config"
	"openpitrix.io/newbilling/pkg/constants"
	"openpitrix.io/newbilling/pkg/global"
	"openpitrix.io/newbilling/pkg/logger"
)

func WebService() *restful.WebService {
	restful.RegisterEntityAccessor(constants.MIME_MERGEPATCH, restful.NewEntityAccessorJSON(restful.MIME_JSON))

	ws := new(restful.WebService)
	ws.Path("/api/v1alpha1").Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).Produces(restful.MIME_JSON)

	tags := []string{"Resource"}

	ws.Route(ws.GET("/nodes").To(DescribeNodes).
		Doc("Describe Nodes Types").
		Param(ws.QueryParameter("watch", "watch resource, true/false.").DataType("bool").DefaultValue("false").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	return ws
}

var Container = restful.DefaultContainer

func Run() {
	Container.Add(WebService())
	enableCORS()

	global.GetInstance()

	cfg := config.GetInstance()
	apiPort, _ := strconv.Atoi(cfg.App.ApiPort)
	listen := fmt.Sprintf(":%d", apiPort)

	logger.Info(nil, "%+v", http.ListenAndServe(listen, nil))
}

func enableCORS() {
	// Optionally, you may need to enable CORS for the UI to work.
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		CookiesAllowed: false,
		AllowedDomains: []string{"*"},
		Container:      Container}
	Container.Filter(cors.Filter)
}
