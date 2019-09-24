// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package apiserver

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"context"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"

	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/constants"
	"openpitrix.io/scheduler/pkg/global"
	"openpitrix.io/scheduler/pkg/logger"
)

func WebService() *restful.WebService {
	restful.RegisterEntityAccessor(constants.MIME_MERGEPATCH, restful.NewEntityAccessorJSON(restful.MIME_JSON))

	ws := new(restful.WebService)
	ws.Path("/api/v1alpha1").Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).Produces(restful.MIME_JSON)

	tags := []string{"Resource"}

	ws.Route(ws.POST("/nodes/{node_name}").To(CreateNode).
		Doc("Create Node").
		Param(ws.PathParameter("node_name", "Specify node").DataType("string").Required(true).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	ws.Route(ws.GET("/nodes/").To(DescribeNodes).
		Doc("Describe Nodes").
		Param(ws.QueryParameter("watch", "watch resource, true/false.").DataType("bool").DefaultValue("false").Required(false)).
		Param(ws.QueryParameter("filter", "filter, eg. group=abc.").DataType("string").DefaultValue("").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	ws.Route(ws.GET("/nodes/{node_name}").To(DescribeNode).
		Doc("Describe Nodes").
		Param(ws.QueryParameter("watch", "watch resource, true/false.").DataType("bool").DefaultValue("false").Required(false)).
		Param(ws.QueryParameter("filter", "filter, eg. group=abc.").DataType("string").DefaultValue("").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	tags = []string{"Task"}

	ws.Route(ws.POST("/tasks/{task_name}").To(CreateTask).
		Doc("Create Task").
		Param(ws.PathParameter("task_name", "Specify task").DataType("string").Required(true).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	ws.Route(ws.GET("/tasks/").To(DescribeTasks).
		Doc("Describe Tasks").
		Param(ws.QueryParameter("watch", "watch resource, true/false.").DataType("bool").DefaultValue("false").Required(false)).
		Param(ws.QueryParameter("filter", "filter, eg. group=abc.").DataType("string").DefaultValue("").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	tags = []string{"Job"}

	ws.Route(ws.POST("/jobs/{job_name}").To(CreateJob).
		Doc("Create Job").
		Param(ws.PathParameter("job_name", "Specify job").DataType("string").Required(true).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	ws.Route(ws.GET("/jobs/").To(DescribeJobs).
		Doc("Describe Jobs").
		Param(ws.QueryParameter("watch", "watch resource, true/false.").DataType("bool").DefaultValue("false").Required(false)).
		Param(ws.QueryParameter("filter", "filter, eg. group=abc.").DataType("string").DefaultValue("").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, constants.MIME_MERGEPATCH).
		Produces(restful.MIME_JSON))

	return ws
}

var Container = restful.DefaultContainer

func stream(w http.ResponseWriter) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Server does not support Flusher!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	timer := time.NewTicker(time.Second * 2)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			logger.Info(nil, "Write")
			w.Write([]byte("timer\n"))
			flusher.Flush()
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	stream(w)
}

func test() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":4567", nil)
}

func Run() {
	Container.Add(WebService())
	enableCORS()

	global.GetInstance()

	go watchGlobal("nodes/")

	go test()

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

func watchGlobal(key string) {
	e := global.GetInstance().GetEtcd()
	watchRes := e.Watch(context.Background(), key, clientv3.WithPrefix())

	for res := range watchRes {
		for _, ev := range res.Events {
			if ev.Type == mvccpb.PUT {
				logger.Info(nil, "watchGlobal got put event [%s] [%s]", string(ev.Kv.Key), string(ev.Kv.Value))
			} else if ev.Type == mvccpb.DELETE {
				logger.Info(nil, "watchGlobal got delete event [%s] [%s]", string(ev.Kv.Key), string(ev.Kv.Value))
			}
		}
	}
}
