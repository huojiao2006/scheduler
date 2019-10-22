// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"sync"

	"github.com/robfig/cron/v3"

	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type CronRunners struct {
	sync.RWMutex
	Map map[string]*CronRunner
}

type Controller struct {
	jobWatcher  *JobWatcher
	cronWatcher *CronWatcher
	cronCore    *cron.Cron
	cronRunners *CronRunners
}

func NewController() *Controller {
	ct := &Controller{
		jobWatcher:  NewJobWatcher("Status=Created"),
		cronWatcher: NewCronWatcher(""),
		cronCore:    cron.New(),
		cronRunners: &CronRunners{Map: make(map[string]*CronRunner)},
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
	ct.cronRunners.Lock()
	_, ok := ct.cronRunners.Map[cronInfo.Name]
	ct.cronRunners.Unlock()
	if ok {
		logger.Error(nil, "Controller cronRun error: cron %s already exists", cronInfo.Name)
		return
	}

	cronRunner := NewCronRunner(ct.cronCore, cronInfo)

	ct.cronRunners.Lock()
	ct.cronRunners.Map[cronInfo.Name] = cronRunner
	ct.cronRunners.Unlock()

	cronRunner.Run()
}

func (ct *Controller) scheduleCron(cronInfo models.CronInfo) {
	go ct.cronRun(cronInfo)
}

func (ct *Controller) stopCron(name string) {
	ct.cronRunners.Lock()
	cronRunner, ok := ct.cronRunners.Map[name]
	ct.cronRunners.Unlock()
	if !ok {
		logger.Error(nil, "Controller stopCron error: cron does not exist")
	}

	cronRunner.Stop()

	ct.cronRunners.Lock()
	delete(ct.cronRunners.Map, name)
	ct.cronRunners.Unlock()
}

func (ct *Controller) scheduleCronLoop() {
	for {
		select {
		case cronEvent := <-ct.cronWatcher.cronChan:
			logger.Info(nil, "scheduleCron %v", cronEvent)

			switch cronEvent.Event {
			case "ADD":
				ct.scheduleCron(cronEvent.CronInfo)
			case "DELETE":
				ct.stopCron(cronEvent.CronInfo.Name)
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
