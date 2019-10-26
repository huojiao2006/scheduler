// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package nodeagent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/config"
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

	cfg := config.GetInstance()

	url := fmt.Sprintf("http://%s:%s/api/v1alpha1", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)

	writer.WriteAPIServer(url, "tasks", taskInfo.Name, string(value))
}

func (na *NodeAgent) runCmd(app string, args []string) {
	cmd := exec.Command(app, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Start()
	cmd.Wait()
}

func (na *NodeAgent) runTask(taskInfo models.TaskInfo) {
	//1.Start running task
	taskInfo.Status = "Running"
	taskInfo.StartTime = time.Now()
	na.updateTask(taskInfo)

	//2.Running task
	logger.Debug(nil, "Run task %v", taskInfo.Cmd)
	na.runCmd(taskInfo.Cmd[0], taskInfo.Cmd[1:])

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
