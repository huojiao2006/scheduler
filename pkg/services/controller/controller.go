// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
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

func (ct *Controller) jobRun(jobInfo models.JobInfo) {
	jobRunner := NewJobRunner(jobInfo)

	jobRunner.Run()
}

func (ct *Controller) scheduleJob(jobInfo models.JobInfo) {
	go ct.jobRun(jobInfo)
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
	go ct.jobWatcher.Run()
	ct.scheduleLoop()
}
