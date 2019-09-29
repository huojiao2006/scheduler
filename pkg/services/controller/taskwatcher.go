// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"
	"fmt"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type TaskWatcher struct {
	Owner        string
	taskChan     chan models.TaskInfo
	taskInformer *informer.Informer
}

func NewTaskWatcher(owner string) *TaskWatcher {
	tw := &TaskWatcher{
		Owner:    owner,
		taskChan: make(chan models.TaskInfo, 100),
	}

	return tw
}

func (tw *TaskWatcher) notifyTask(value []byte) {
	taskInfo := models.TaskInfo{}

	err := json.Unmarshal(value, &taskInfo)
	if err != nil {
		logger.Error(nil, "Unmarshal TaskInfo error: %v", err)
		return
	}

	tw.taskChan <- taskInfo
}

func (tw *TaskWatcher) watchTasks() {
	url := fmt.Sprintf("http://127.0.0.1:8080/api/v1alpha1/tasks/?watch=true&filter=Owner=%s", tw.Owner)
	tw.taskInformer = informer.NewInformer(url)

	tw.taskInformer.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Debug(nil, "watchTasks added task: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				tw.notifyTask(info.Value)
			} else {
				logger.Error(nil, "watchTasks data error")
			}
		},
		DeleteFunc: func(obj interface{}) {
			logger.Debug(nil, "watchTasks deleted task: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				tw.notifyTask(info.Value)
			} else {
				logger.Error(nil, "watchTasks data error")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Debug(nil, "watchTasks updated task: %v", newObj)

			info, ok := (oldObj).(models.Info)
			if ok {
				tw.notifyTask(info.Value)
			} else {
				logger.Error(nil, "watchTasks data error")
			}
		},
	})

	tw.taskInformer.Start()
}

func (tw *TaskWatcher) Run() {
	tw.watchTasks()
}

func (tw *TaskWatcher) Stop() {
	tw.taskInformer.Stop()
	close(tw.taskChan)
}
