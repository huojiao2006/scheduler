// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/logger"
)

func exitHandler() {
	c := make(chan os.Signal)

	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				ExitFunc()
			case syscall.SIGUSR1:
			case syscall.SIGUSR2:
			default:
			}
		}
	}()
}

func ExitFunc() {
	os.Exit(0)
}

type ResourceEventHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj, newObj interface{})
	OnDelete(obj interface{})
}

// ResourceEventHandlerFuncs is an adaptor to let you easily specify as many or
// as few of the notification functions as you want while still implementing
// ResourceEventHandler.
type ResourceEventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
	DeleteFunc func(obj interface{})
}

// OnAdd calls AddFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnAdd(obj interface{}) {
	if r.AddFunc != nil {
		r.AddFunc(obj)
	}
}

// OnUpdate calls UpdateFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnUpdate(oldObj, newObj interface{}) {
	if r.UpdateFunc != nil {
		r.UpdateFunc(oldObj, newObj)
	}
}

// OnDelete calls DeleteFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnDelete(obj interface{}) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

type Informer struct {
	url     string
	handler ResourceEventHandler
}

type Info struct {
	Value          []byte `json:"Value"`
	CreateRevision int64  `json:"CreateRevision"`
	ModRevision    int64  `json:"ModRevision"`
	Version        int64  `json:"Version"`
}

type Event struct {
	Event string `json:"Event"`
	Value Info   `json:"Value"`
}

func (i *Informer) watch() {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost:   10000,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	request, err := http.NewRequest("GET", i.url, nil)
	if err != nil {
		logger.Error(nil, "SendMetricRequest NewRequest error:", err)
		return
	}

	params := request.URL.Query()
	params.Add("watch", "true")
	request.URL.RawQuery = params.Encode()

	logger.Debug(nil, "watch %s", request.URL.RawQuery)

	response, err := client.Do(request)
	if err != nil {
		logger.Error(nil, "watch get error:", err)
	} else {
		defer response.Body.Close()

		reader := bufio.NewReader(response.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				logger.Error(nil, "watch read error:", err.Error())
			} else {
				//logger.Info(nil, "%s", string(line))
				var event Event
				err := json.Unmarshal(line, &event)
				if err != nil {
					logger.Error(nil, "watch unmarshal error [%v]", err)
					continue
				}
				switch event.Event {
				case "ADD":
					i.handler.OnAdd(event.Value)
				case "MODIFY":
					i.handler.OnUpdate(event.Value, event.Value)
				case "DELETE":
					i.handler.OnDelete(event.Value)
				}
			}
		}

		if err != nil {
			logger.Error(nil, "watch read error:", err.Error())
		}
	}
}

func (i *Informer) AddEventHandler(handler ResourceEventHandler) {
	i.handler = handler
}

func (i *Informer) Start() {
	go i.watch()
}

func NewInformer(url string) *Informer {
	return &Informer{
		url: url,
	}
}

func main() {
	exitHandler()

	config.GetInstance().LoadConf()

	cfg := config.GetInstance()

	url := fmt.Sprintf("http://%s:%s/api/v1alpha1/nodes/node1", cfg.ApiServer.ApiHost, cfg.ApiServer.ApiPort)
	informer := NewInformer(url)

	informer.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logger.Info(nil, "Added: %v", obj)
		},
		DeleteFunc: func(obj interface{}) {
			logger.Info(nil, "Deleted: %v", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			logger.Info(nil, "Updated: %v", newObj)
		},
	})

	informer.Start()

	for {
		logger.Debug(nil, "running...")
		time.Sleep(time.Second)
	}
}
