// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
	"sync"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type NodeMap struct {
	sync.RWMutex
	Map map[string]string
}

type NodeWatcher struct {
	nodeMap *NodeMap
}

func NewNodeWatcher() *NodeWatcher {
	nw := &NodeWatcher{
		nodeMap: &NodeMap{Map: make(map[string]string)},
	}

	return nw
}

func (nw *NodeWatcher) addNode(node string) {
	nw.nodeMap.Lock()
	nw.nodeMap.Map[node] = "Alive"
	nw.nodeMap.Unlock()
}

func (nw *NodeWatcher) deleteNode(node string) {
	nw.nodeMap.Lock()
	delete(nw.nodeMap.Map, node)
	nw.nodeMap.Unlock()
}

func (nw *NodeWatcher) watchNodes() {
	nodeInformer := informer.NewInformer("http://127.0.0.1:8080/api/v1alpha1/nodes/")

	nodeInformer.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "watchNodes added node: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				nw.addNode(info.Key)
			} else {
				logger.Info(nil, "watchNodes data error")
			}
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "watchNodes deleted node: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				nw.deleteNode(info.Key)
			} else {
				logger.Info(nil, "watchNodes data error")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "watchNodes updated node: %v", newObj)
		},
	})

	nodeInformer.Start()
}

func (nw *NodeWatcher) SelectNode() string {
	return ""
}

func (nw *NodeWatcher) Run() {
	nw.watchNodes()
}
