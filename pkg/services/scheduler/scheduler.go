// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
)

type Scheduler struct {
	nodeWatcher *NodeWatcher
}

func NewScheduler() *Scheduler {
	sc := &Scheduler{
		nodeWatcher: NewNodeWatcher(),
	}
	return sc
}

func Init() *Scheduler {
	scheduler := NewScheduler()

	return scheduler
}

func (sc *Scheduler) Run() {
	sc.nodeWatcher.Run()
}
