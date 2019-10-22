// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/constants"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
	"openpitrix.io/scheduler/pkg/util/idutil"
)

type CronRunner struct {
	entryId    cron.EntryID
	cronCore   *cron.Cron
	cronInfo   models.CronInfo
	jobWatcher *JobWatcher
	stopChan   chan string
}

func NewJobId() string {
	return idutil.GetUuid(constants.JobIdPrefix)
}

func (cr *CronRunner) cronFunc() {
	jobId := NewJobId()

	jobInfo := models.JobInfo{
		Name:   jobId,
		Owner:  cr.cronInfo.Name,
		Status: "Created",
	}

	cr.createJob(jobInfo)
}

func (cr *CronRunner) updateCron(cronInfo models.CronInfo) {
	value, err := json.Marshal(cronInfo)
	if err != nil {
		logger.Error(nil, "updateCron marshal cron info error [%v]", err)
		return
	}

	info := models.APIInfo{
		Info: string(value),
		TTL:  0,
	}

	value, err = json.Marshal(info)
	if err != nil {
		logger.Error(nil, "updateCron marshal info error [%v]", err)
		return
	}

	cfg := config.GetInstance()

	url := fmt.Sprintf("http://%s:%s/api/v1alpha1", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	writer.WriteAPIServer(url, "crons", cronInfo.Name, string(value))
}

func (cr *CronRunner) createJob(jobInfo models.JobInfo) {
	value, err := json.Marshal(jobInfo)
	if err != nil {
		logger.Error(nil, "createJob marshal job info error [%v]", err)
		return
	}

	info := models.APIInfo{
		Info: string(value),
		TTL:  0,
	}

	value, err = json.Marshal(info)
	if err != nil {
		logger.Error(nil, "createJob marshal info error [%v]", err)
		return
	}

	cfg := config.GetInstance()

	url := fmt.Sprintf("http://%s:%s/api/v1alpha1", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	writer.WriteAPIServer(url, "jobs", jobInfo.Name, string(value))
}

func NewCronRunner(cronCore *cron.Cron, cronInfo models.CronInfo) *CronRunner {
	cr := &CronRunner{
		cronCore:   cronCore,
		cronInfo:   cronInfo,
		jobWatcher: NewJobWatcher(fmt.Sprintf("Owner=%s", cronInfo.Name)),
		stopChan:   make(chan string, 1),
	}
	return cr
}

func (cr *CronRunner) jobMonitor() {
	defer close(cr.stopChan)

	cronInfoMonitor := models.CronInfo{
		Name:             cr.cronInfo.Name,
		Script:           cr.cronInfo.Script,
		Owner:            cr.cronInfo.Owner,
		Status:           cr.cronInfo.Status,
		LastScheduleTime: cr.cronInfo.LastScheduleTime,
	}

	for {
		select {
		case <-cr.stopChan:
			return
		case jobEvent := <-cr.jobWatcher.jobChan:
			logger.Info(nil, "jobMonitor %v", jobEvent)
			switch jobEvent.JobInfo.Status {
			case "Running":
				cronInfoMonitor.Status = "Active"
				cr.updateCron(cronInfoMonitor)
			case "Completed":
				cronInfoMonitor.Status = ""
				cronInfoMonitor.LastScheduleTime = time.Now()
				cr.updateCron(cronInfoMonitor)
			}
		}
	}
}

func (cr *CronRunner) Run() {
	logger.Info(nil, "Cron Runner Starting Cron[%v]", cr.cronInfo)

	cr.entryId, _ = cr.cronCore.AddFunc(cr.cronInfo.Script, cr.cronFunc)

	logger.Info(nil, "Cron Runner Started Cron[%d]", cr.entryId)

	cr.jobWatcher.watchJobs()

	cr.jobMonitor()

	logger.Info(nil, "Cron Runner Stopped Cron %d", cr.entryId)
}

func (cr *CronRunner) Stop() {
	logger.Info(nil, "Cron Runner Stopping Cron %d", cr.entryId)

	cr.cronCore.Remove(cr.entryId)
	cr.stopChan <- "stop"
}
