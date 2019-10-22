// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"
	"fmt"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type JobWatcher struct {
	filter  string
	jobChan chan models.JobEvent
}

func NewJobWatcher(filter string) *JobWatcher {
	jw := &JobWatcher{
		filter:  filter,
		jobChan: make(chan models.JobEvent, 100),
	}

	return jw
}

func (jw *JobWatcher) scheduleJob(event string, value []byte) {
	jobInfo := models.JobInfo{}

	err := json.Unmarshal(value, &jobInfo)
	if err != nil {
		logger.Error(nil, "Unmarshal JobInfo error: %v", err)
		return
	}

	jobEvent := models.JobEvent{
		Event:   event,
		JobInfo: jobInfo,
	}

	jw.jobChan <- jobEvent
}

func (jw *JobWatcher) watchJobs() {
	cfg := config.GetInstance()

	informerURL := fmt.Sprintf("http://%s:%s/api/v1alpha1/jobs/?watch=true", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	if jw.filter == "" {
		informerURL = fmt.Sprintf("http://%s:%s/api/v1alpha1/jobs/?watch=true", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	} else {
		informerURL = fmt.Sprintf("http://%s:%s/api/v1alpha1/jobs/?watch=true&filter=%s", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort, jw.filter)
	}

	jobInformar := informer.NewInformer(informerURL)

	jobInformar.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "watchJobs added job: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				jw.scheduleJob("ADD", info.Value)
			} else {
				logger.Error(nil, "watchJobs data error")
			}
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "watchJobs deleted job: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				jw.scheduleJob("DELETE", info.Value)
			} else {
				logger.Error(nil, "watchJobs data error")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "watchJobs updated job: %v", newObj)

			info, ok := (newObj).(models.Info)
			if ok {
				jw.scheduleJob("MODIFY", info.Value)
			} else {
				logger.Error(nil, "watchJobs data error")
			}
		},
	})

	jobInformar.Start()
}

func (jw *JobWatcher) Run() {
	jw.watchJobs()
}
