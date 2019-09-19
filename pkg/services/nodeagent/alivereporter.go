// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package nodeagent

import (
	"encoding/json"
	"time"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type AliveReporter struct {
	nodeAgent *NodeAgent
}

func NewAliveReporter() *AliveReporter {
	ar := &AliveReporter{}

	return ar
}

func (ar *AliveReporter) SetNodeAgent(nodeAgent *NodeAgent) {
	ar.nodeAgent = nodeAgent
}

func (ar *AliveReporter) doHeartBeat() {
	nodeInfo := models.NodeInfo{
		Info: "",
		TTL:  60,
	}

	value, err := json.Marshal(nodeInfo)
	if err != nil {
		logger.Error(nil, "doHeartBeat marshal node info error [%v]", err)
		return
	}

	writer.WriteAPIServer("http://127.0.0.1:8080/api/v1alpha1", "nodes", ar.nodeAgent.HostName, string(value))
}

func (ar *AliveReporter) HeartBeat() {
	ar.doHeartBeat()

	timer := time.NewTicker(time.Second * 20)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			ar.doHeartBeat()
		}
	}
}
