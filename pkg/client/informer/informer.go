package informer

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

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
	url      string
	handler  ResourceEventHandler
	stopChan chan string
}

var client = &http.Client{
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

func (i *Informer) watch() {
	defer close(i.stopChan)

	request, err := http.NewRequest("GET", i.url, nil)
	if err != nil {
		logger.Error(nil, "Informer watch NewRequest error:", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelRoutine := make(chan struct{})
	defer close(cancelRoutine)

	request = request.WithContext(ctx)

	go func() {
		for {
			select {
			case <-i.stopChan:
				cancel()
				return
			case <-cancelRoutine:
				return
			}
		}
	}()

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
				break
			} else {
				var event models.Event
				err := json.Unmarshal(line, &event)
				if err != nil {
					logger.Error(nil, "watch unmarshal error [%v]", err)
					continue
				}
				switch event.Event {
				case "ADD":
					i.handler.OnAdd(event.Data)
				case "MODIFY":
					i.handler.OnUpdate(event.Data, event.Data)
				case "DELETE":
					i.handler.OnDelete(event.Data)
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

func (i *Informer) Stop() {
	i.stopChan <- "close"
}

func NewInformer(url string) *Informer {
	return &Informer{
		url:      url,
		stopChan: make(chan string, 1),
	}
}
