// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
	"encoding/json"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type TaskWatcher struct {
	taskChan chan models.TaskInfo
}

func NewTaskWatcher() *TaskWatcher {
	tw := &TaskWatcher{
		taskChan: make(chan models.TaskInfo, 100),
	}

	return tw
}

func (tw *TaskWatcher) scheduleTask(value []byte) {
	taskInfo := models.TaskInfo{}

	err := json.Unmarshal(value, &taskInfo)
	if err != nil {
		logger.Error(nil, "Unmarshal TaskInfo error: %v", err)
		return
	}

	tw.taskChan <- taskInfo
}

func (tw *TaskWatcher) watchTasks() {
	taskInformar := informer.NewInformer("http://127.0.0.1:8080/api/v1alpha1/tasks/?watch=true&filter=Node=,Status=Pending")

	taskInformar.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "watchTasks added task: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				tw.scheduleTask(info.Value)
			} else {
				logger.Info(nil, "watchTasks data error")
			}
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "watchTasks deleted task: %v", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "watchTasks updated task: %v", newObj)
		},
	})

	taskInformar.Start()
}

func (tw *TaskWatcher) Run() {
	tw.watchTasks()
}
