// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
	"encoding/json"
	"fmt"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type Scheduler struct {
	nodeWatcher *NodeWatcher
	taskWatcher *TaskWatcher
}

func NewScheduler() *Scheduler {
	sc := &Scheduler{
		nodeWatcher: NewNodeWatcher(),
		taskWatcher: NewTaskWatcher(),
	}
	return sc
}

func Init() *Scheduler {
	scheduler := NewScheduler()

	return scheduler
}

func (sc *Scheduler) updateTask(taskInfo models.TaskInfo) {
	value, err := json.Marshal(taskInfo)
	if err != nil {
		logger.Error(nil, "updateTask marshal task info error [%v]", err)
		return
	}

	info := models.APIInfo{
		Info: string(value),
		TTL:  0,
	}

	value, err = json.Marshal(info)
	if err != nil {
		logger.Error(nil, "updateTask marshal info error [%v]", err)
		return
	}

	cfg := config.GetInstance()

	url := fmt.Sprintf("http://%s:%s/api/v1alpha1", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	writer.WriteAPIServer(url, "tasks", taskInfo.Name, string(value))
}

func (sc *Scheduler) scheduleTask(taskInfo models.TaskInfo) {
	//Choose node randomly and assign task
	nodeSelected := sc.nodeWatcher.SelectNode()

	if "" == nodeSelected {
		logger.Info(nil, "Scheduler has no node to schedule")
		return
	}

	taskInfo.Node = nodeSelected
	taskInfo.Status = "Scheduled"

	sc.updateTask(taskInfo)
}

func (sc *Scheduler) scheduleLoop() {
	for {
		select {
		case taskInfo := <-sc.taskWatcher.taskChan:
			logger.Debug(nil, "scheduleTask %v", taskInfo)

			sc.scheduleTask(taskInfo)
		}
	}
}

func (sc *Scheduler) Run() {
	go sc.nodeWatcher.Run()
	go sc.taskWatcher.Run()
	sc.scheduleLoop()
}
