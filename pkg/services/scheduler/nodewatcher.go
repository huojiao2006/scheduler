// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package scheduler

import (
	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/logger"
)

type NodeWatcher struct {
	nodeMap map[string]string
}

func NewNodeWatcher() *NodeWatcher {
	nw := &NodeWatcher{
		nodeMap: make(map[string]string),
	}

	return nw
}

func (nw *NodeWatcher) watchNodes() {
	nodeInformer := informer.NewInformer("http://127.0.0.1:8080/api/v1alpha1/nodes/")

	nodeInformer.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "watchNodes added node: %v", obj)
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "watchNodes deleted node: %v", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "watchNodes updated node: %v", newObj)
		},
	})

	nodeInformer.Start()
}

func (nw *NodeWatcher) Run() {
	nw.watchNodes()
}
