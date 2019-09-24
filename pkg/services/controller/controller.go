// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type Controller struct {
	jobWatcher *JobWatcher
}

func NewController() *Controller {
	ct := &Controller{
		jobWatcher: NewJobWatcher(),
	}
	return ct
}

func Init() *Controller {
	controller := NewController()

	return controller
}

func (sc *Controller) updateJob(jobInfo models.JobInfo) {
	value, err := json.Marshal(jobInfo)
	if err != nil {
		logger.Error(nil, "updateJob marshal job info error [%v]", err)
		return
	}

	info := models.APIInfo{
		Info: string(value),
		TTL:  0,
	}

	value, err = json.Marshal(info)
	if err != nil {
		logger.Error(nil, "updateJob marshal info error [%v]", err)
		return
	}

	writer.WriteAPIServer("http://127.0.0.1:8080/api/v1alpha1", "jobs", jobInfo.Name, string(value))
}

func (ct *Controller) scheduleJob(jobInfo models.JobInfo) {
}

func (ct *Controller) scheduleLoop() {
	for {
		select {
		case jobInfo := <-ct.jobWatcher.jobChan:
			logger.Debug(nil, "scheduleJob %v", jobInfo)

			ct.scheduleJob(jobInfo)
		}
	}
}

func (ct *Controller) Run() {
	ct.jobWatcher.Run()
}
