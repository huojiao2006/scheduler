// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package nodeagent

import (
	"os"

	"openpitrix.io/scheduler/pkg/logger"
)

type NodeAgent struct {
	HostName      string
	aliveReporter *AliveReporter
}

func NewNodeAgent(aliveReporter *AliveReporter) *NodeAgent {
	host, err := os.Hostname()
	if err != nil {
		logger.Error(nil, "NewNodeAgent get host name error: %s", err)
	}

	na := &NodeAgent{
		HostName:      host,
		aliveReporter: aliveReporter,
	}
	return na
}

func Init() *NodeAgent {
	aliveReporter := NewAliveReporter()

	nodeAgent := NewNodeAgent(aliveReporter)

	aliveReporter.SetNodeAgent(nodeAgent)

	return nodeAgent
}

func (na *NodeAgent) Run() {
	na.aliveReporter.HeartBeat()
}
