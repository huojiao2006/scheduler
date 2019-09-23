// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
	"math/rand"
	"strings"
	"sync"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type NodeStorage struct {
	sync.RWMutex
	Map  map[string]int
	List []string
}

type NodeWatcher struct {
	nodeStorage *NodeStorage
}

func NewNodeWatcher() *NodeWatcher {
	nw := &NodeWatcher{
		nodeStorage: &NodeStorage{Map: make(map[string]int), List: []string{}},
	}

	return nw
}

func (nw *NodeWatcher) addNode(node string) {
	node = strings.TrimPrefix(node, "nodes/")
	nw.nodeStorage.Lock()
	if _, ok := nw.nodeStorage.Map[node]; ok {
		logger.Error(nil, "addNode error: node already registered")
	} else {
		nw.nodeStorage.List = append(nw.nodeStorage.List, node)
		nw.nodeStorage.Map[node] = len(nw.nodeStorage.List) - 1
	}
	nw.nodeStorage.Unlock()
}

func (nw *NodeWatcher) deleteNode(node string) {
	node = strings.TrimPrefix(node, "nodes/")
	nw.nodeStorage.Lock()
	if index, ok := nw.nodeStorage.Map[node]; ok {
		nw.nodeStorage.List = append(nw.nodeStorage.List[:index], nw.nodeStorage.List[index+1:]...)
		delete(nw.nodeStorage.Map, node)
	} else {
		logger.Error(nil, "deleteNode error: node not registered")
	}
	nw.nodeStorage.Unlock()
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
	nw.nodeStorage.Lock()
	defer nw.nodeStorage.Unlock()

	if len(nw.nodeStorage.List) == 0 {
		logger.Info(nil, "SelectNode has no node to schedule")
		return ""
	}

	return nw.nodeStorage.List[rand.Intn(len(nw.nodeStorage.List))]
}

func (nw *NodeWatcher) Run() {
	nw.watchNodes()
}
