// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package nodeagent

import (
	"encoding/json"
	"os"
	"time"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type NodeAgent struct {
	HostName      string
	aliveReporter *AliveReporter
	taskWatcher   *TaskWatcher
}

func NewNodeAgent() *NodeAgent {
	host, err := os.Hostname()
	if err != nil {
		logger.Error(nil, "NewNodeAgent get host name error: %s", err)
	}

	na := &NodeAgent{
		HostName:      host,
		aliveReporter: NewAliveReporter(),
		taskWatcher:   NewTaskWatcher(host),
	}
	return na
}

func Init() *NodeAgent {
	nodeAgent := NewNodeAgent()

	nodeAgent.aliveReporter.SetNodeAgent(nodeAgent)

	return nodeAgent
}

func (na *NodeAgent) updateTask(taskInfo models.TaskInfo) {
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

	writer.WriteAPIServer("http://127.0.0.1:8080/api/v1alpha1", "tasks", taskInfo.Name, string(value))
}

func (na *NodeAgent) runTask(taskInfo models.TaskInfo) {
	//1.Start running task
	taskInfo.Status = "Running"
	taskInfo.StartTime = time.Now()
	na.updateTask(taskInfo)

	//2.Running task
	time.Sleep(time.Second)

	//3.Complete task
	taskInfo.Status = "Completed"
	taskInfo.CompleteTime = time.Now()
	na.updateTask(taskInfo)
}

func (na *NodeAgent) runLoop() {
	for {
		select {
		case taskInfo := <-na.taskWatcher.taskChan:
			logger.Debug(nil, "runTask %v", taskInfo)

			go na.runTask(taskInfo)
		}
	}
}

func (na *NodeAgent) Run() {
	go na.aliveReporter.HeartBeat()
	go na.taskWatcher.Run()
	na.runLoop()
}
