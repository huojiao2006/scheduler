// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"openpitrix.io/scheduler/pkg/client/informer"
	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

type CronWatcher struct {
	filter   string
	cronChan chan models.CronEvent
}

func NewCronWatcher(filter string) *CronWatcher {
	cw := &CronWatcher{
		filter:   filter,
		cronChan: make(chan models.CronEvent, 100),
	}

	return cw
}

func (cw *CronWatcher) scheduleCron(event string, key string, value []byte) {
	cronInfo := models.CronInfo{}

	if event == "DELETE" {
		cronInfo.Name = strings.TrimPrefix(key, "crons/")
	} else {
		err := json.Unmarshal(value, &cronInfo)
		if err != nil {
			logger.Error(nil, "Unmarshal CronInfo error: %v", err)
			return
		}
	}

	cronEvent := models.CronEvent{
		Event:    event,
		CronInfo: cronInfo,
	}

	cw.cronChan <- cronEvent
}

func (cw *CronWatcher) watchCrons() {
	cfg := config.GetInstance()

	informerURL := fmt.Sprintf("http://%s:%s/api/v1alpha1/crons/?watch=true", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	if cw.filter == "" {
		informerURL = fmt.Sprintf("http://%s:%s/api/v1alpha1/crons/?watch=true", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	} else {
		informerURL = fmt.Sprintf("http://%s:%s/api/v1alpha1/crons/?watch=true&filter=%s", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort, cw.filter)
	}

	cronInformer := informer.NewInformer(informerURL)

	cronInformer.AddEventHandler(informer.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "watchCrons added cron: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				cw.scheduleCron("ADD", info.Key, info.Value)
			} else {
				logger.Error(nil, "watchCrons data error")
			}
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "watchCrons deleted cron: %v", obj)

			info, ok := (obj).(models.Info)
			if ok {
				cw.scheduleCron("DELETE", info.Key, info.Value)
			} else {
				logger.Error(nil, "watchCrons data error")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "watchCrons updated cron: %v", newObj)

			info, ok := (newObj).(models.Info)
			if ok {
				cw.scheduleCron("MODIFY", info.Key, info.Value)
			} else {
				logger.Error(nil, "watchCrons data error")
			}
		},
	})

	cronInformer.Start()
}

func (cw *CronWatcher) Run() {
	cw.watchCrons()
}
