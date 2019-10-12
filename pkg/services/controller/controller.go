// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"github.com/robfig/cron"

	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type Controller struct {
	jobWatcher  *JobWatcher
	cronWatcher *CronWatcher
	cronCore    *cron.Cron
}

func NewController() *Controller {
	ct := &Controller{
		jobWatcher:  NewJobWatcher("Status=Created"),
		cronWatcher: NewCronWatcher(""),
		cronCore:    cron.New(),
	}

	ct.cronCore.Start()

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

func (ct *Controller) scheduleJobLoop() {
	for {
		select {
		case jobEvent := <-ct.jobWatcher.jobChan:
			logger.Debug(nil, "scheduleJob %v", jobEvent)

			if jobEvent.Event == "ADD" {
				ct.scheduleJob(jobEvent.JobInfo)
			}
		}
	}
}

func (ct *Controller) cronRun(cronInfo models.CronInfo) {
	cronRunner := NewCronRunner(ct.cronCore, cronInfo)

	cronRunner.Run()
}

func (ct *Controller) scheduleCron(cronInfo models.CronInfo) {
	go ct.cronRun(cronInfo)
}

func (ct *Controller) scheduleCronLoop() {
	for {
		select {
		case cronEvent := <-ct.cronWatcher.cronChan:
			logger.Debug(nil, "scheduleCron %v", cronEvent)

			if cronEvent.Event == "ADD" {
				ct.scheduleCron(cronEvent.CronInfo)
			}
		}
	}
}

func (ct *Controller) Run() {
	go ct.jobWatcher.Run()
	go ct.cronWatcher.Run()
	go ct.scheduleJobLoop()
	ct.scheduleCronLoop()
}
