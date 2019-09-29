// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"
	"sync"
	"time"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/constants"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
	"openpitrix.io/scheduler/pkg/util/idutil"
)

type JobRunner struct {
	jobInfo     models.JobInfo
	taskWatcher *TaskWatcher
}

func NewTaskId() string {
	return idutil.GetUuid(constants.TaskIdPrefix)
}

func (jr *JobRunner) updateJob(jobInfo models.JobInfo) {
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

func (jr *JobRunner) createTask(taskInfo models.TaskInfo) {
	value, err := json.Marshal(taskInfo)
	if err != nil {
		logger.Error(nil, "createTask marshal task info error [%v]", err)
		return
	}

	info := models.APIInfo{
		Info: string(value),
		TTL:  0,
	}

	value, err = json.Marshal(info)
	if err != nil {
		logger.Error(nil, "createTask marshal info error [%v]", err)
		return
	}

	writer.WriteAPIServer("http://127.0.0.1:8080/api/v1alpha1", "tasks", taskInfo.Name, string(value))
}

func NewJobRunner(jobInfo models.JobInfo) *JobRunner {
	jr := &JobRunner{
		jobInfo:     jobInfo,
		taskWatcher: NewTaskWatcher(jobInfo.Name),
	}
	return jr
}

func (jr *JobRunner) taskMonitor(wg *sync.WaitGroup) {
	jobInfoNew := models.JobInfo{
		Name:  jr.jobInfo.Name,
		Owner: jr.jobInfo.Owner,
	}

	for {
		select {
		case taskInfo := <-jr.taskWatcher.taskChan:
			logger.Info(nil, "taskMonitor %v", taskInfo)
			switch taskInfo.Status {
			case "Running":
				jobInfoNew.Status = "Running"
				jobInfoNew.StartTime = time.Now()
				jr.updateJob(jobInfoNew)
			case "Completed":
				jobInfoNew.Status = "Completed"
				jobInfoNew.CompleteTime = time.Now()
				jr.updateJob(jobInfoNew)
				wg.Done()
				return
			}
		}
	}
}

func (jr *JobRunner) Run() {
	logger.Info(nil, "Job Runner Start Job[%v]", jr.jobInfo)

	taskId := NewTaskId()
	wg := sync.WaitGroup{}

	jr.taskWatcher.watchTasks()
	wg.Add(1)
	go jr.taskMonitor(&wg)

	taskInfo := models.TaskInfo{
		Name:   taskId,
		Owner:  jr.jobInfo.Name,
		Status: "Pending",
	}

	jr.createTask(taskInfo)

	wg.Wait()

	logger.Info(nil, "Job Runner Complete Job[%v]", jr.jobInfo)
}
