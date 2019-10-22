// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package nodeagent

import (
	"encoding/json"
	"fmt"
	"time"

	"openpitrix.io/scheduler/pkg/client/writer"
	"openpitrix.io/scheduler/pkg/config"
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
	nodeInfo := models.APIInfo{
		Info: "",
		TTL:  60,
	}

	value, err := json.Marshal(nodeInfo)
	if err != nil {
		logger.Error(nil, "doHeartBeat marshal node info error [%v]", err)
		return
	}

	cfg := config.GetInstance()

	url := fmt.Sprintf("http://%s:%s/api/v1alpha1", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	writer.WriteAPIServer(url, "nodes", ar.nodeAgent.HostName, string(value))
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
