// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
	"openpitrix.io/scheduler/pkg/logger"
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

func (sc *Scheduler) scheduleLoop() {
	for {
		select {
		case taskInfo := <-sc.taskWatcher.taskChan:
			logger.Debug(nil, "scheduleTask %v", taskInfo)

			//Choose node randomly and assign task
		}
	}
}

func (sc *Scheduler) Run() {
	go sc.nodeWatcher.Run()
	go sc.taskWatcher.Run()
	sc.scheduleLoop()
}
