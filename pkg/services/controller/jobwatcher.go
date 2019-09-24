// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type JobWatcher struct {
	jobChan chan models.JobInfo
}

func NewJobWatcher() *JobWatcher {
	jw := &JobWatcher{
		jobChan: make(chan models.JobInfo, 100),
	}

	return jw
}

func (jw *JobWatcher) scheduleJob(value []byte) {
	jobInfo := models.JobInfo{}

	err := json.Unmarshal(value, &jobInfo)
	if err != nil {
		logger.Error(nil, "Unmarshal JobInfo error: %v", err)
		return
	}

	jw.jobChan <- jobInfo
}

func (jw *JobWatcher) watchJobs() {
	jobInformar := informer.NewInformer("http://127.0.0.1:8080/api/v1alpha1/jobs/?watch=true&filter=Status=Created")

	jobInformar.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "watchJobs added job: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				jw.scheduleJob(info.Value)
			} else {
				logger.Info(nil, "watchJobs data error")
			}
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "watchJobs deleted job: %v", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "watchJobs updated job: %v", newObj)
		},
	})

	jobInformar.Start()
}

func (jw *JobWatcher) Run() {
	jw.watchJobs()
}
