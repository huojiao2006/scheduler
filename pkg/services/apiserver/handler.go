package apiserver

import (
	//"fmt"

	"context"
	"encoding/json"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"

	"github.com/emicklei/go-restful"

	"net/http"

	"openpitrix.io/newbilling/pkg/global"
	"openpitrix.io/newbilling/pkg/logger"
)

func parseBool(input string) bool {
	if input == "true" {
		return true
	} else {
		return false
	}
}

type NodeInfo struct {
	Info string `json:"info"`
}

type Error struct {
	Message string `json:"message" description:"error message"`
}

func Wrap(err error) Error {
	return Error{Message: err.Error()}
}

type Info struct {
	Value          string `json:"Value"`
	CreateRevision int64  `json:"CreateRevision"`
	ModRevision    int64  `json:"ModRevision"`
	Version        int64  `json:"Version"`
}

type Event struct {
	Event string `json:"Event"`
	Value Info   `json:"Value"`
}

type Watcher struct {
	key       string
	storage   []Info
	eventChan chan Event
	stopChan  chan string
}

func NewWatcher(key string, initValue *Info) *Watcher {
	storage := []Info{}

	if initValue != nil {
		storage = append(storage, *initValue)
	}

	return &Watcher{
		key:       key,
		storage:   storage,
		eventChan: make(chan Event, 10),
		stopChan:  make(chan string, 1),
	}
}

func (wc *Watcher) watch() {
	e := global.GetInstance().GetEtcd()

	defer close(wc.eventChan)

	ctx, cancel := context.WithCancel(context.Background())
	cancelRoutine := make(chan struct{})
	defer close(cancelRoutine)

	watchRes := e.Watch(ctx, wc.key, clientv3.WithPrefix())

	go func() {
		for {
			select {
			case <-wc.stopChan:
				cancel()
				return
			case <-cancelRoutine:
				return
			}
		}
	}()

	if len(wc.storage) > 0 {
		wc.eventChan <- Event{"ADD", wc.storage[0]}
	}

	for res := range watchRes {
		for _, ev := range res.Events {
			if ev.Type == mvccpb.PUT {
				logger.Info(nil, "watch got put event [%s] [%s]", string(ev.Kv.Key), string(ev.Kv.Value))
				info := Info{
					Value:          string(ev.Kv.Value),
					CreateRevision: ev.Kv.CreateRevision,
					ModRevision:    ev.Kv.ModRevision,
					Version:        ev.Kv.Version,
				}
				event := "MODIFY"
				if len(wc.storage) == 0 {
					event = "ADD"
				}
				wc.storage = append(wc.storage, info)
				eventInfo := Event{
					Event: event,
					Value: info,
				}
				wc.eventChan <- eventInfo
			} else if ev.Type == mvccpb.DELETE {
				logger.Info(nil, "watch got delete event [%s] [%s]", string(ev.Kv.Key), string(ev.Kv.Value))
				eventInfo := Event{
					Event: "DELETE",
					Value: wc.storage[len(wc.storage)-1],
				}
				wc.eventChan <- eventInfo
				wc.storage = []Info{}
			}
		}
	}

	logger.Info(nil, "watch ended")
}

func (wc *Watcher) stop() {
	wc.stopChan <- "close"
}

func CreateNode(request *restful.Request, response *restful.Response) {
	node := request.PathParameter("node_name")
	nodeInfo := new(NodeInfo)

	err := request.ReadEntity(&nodeInfo)
	if err != nil {
		logger.Debug(nil, "CreateNode request data error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	key := "nodes/" + node

	err = putInfo(key, nodeInfo.Info, 10)
	if err != nil {
		logger.Debug(nil, "CreateNode putInfo error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	logger.Debug(nil, "CreateNode success")

	response.WriteHeaderAndEntity(http.StatusOK, "node")
}

func DescribeNodes(request *restful.Request, response *restful.Response) {
	node := request.PathParameter("node_name")
	watch := parseBool(request.QueryParameter("watch"))

	key := "nodes/" + node

	initValue, err := getInfo(key)
	if err != nil {
		logger.Debug(nil, "CreateNode request data error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	logger.Debug(nil, "DescribeNodes success")

	if watch {
		watcher := NewWatcher(key, initValue)
		go watcher.watch()

		notify := response.CloseNotify()

		for {
			select {
			case event := <-watcher.eventChan:
				response.Write([]byte(formatEvent(event) + "\n"))
				response.Flush()
				logger.Info(nil, "DescribeNodes got event [%v]", event)
			case <-notify:
				watcher.stop()
				logger.Info(nil, "DescribeNodes disconnected")
				return
			}
		}
	} else {
		response.Write([]byte(formatInfo(initValue) + "\n"))
	}
}

func formatEvent(event Event) string {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Error(nil, "formatEvent error [%v]", err)
	}

	return string(eventBytes)
}

func formatInfo(info *Info) string {
	if nil == info {
		return "{}"
	}

	infoBytes, _ := json.Marshal(*info)

	return string(infoBytes)
}

func getInfo(key string) (*Info, error) {
	ctx := context.Background()
	e := global.GetInstance().GetEtcd()

	getResp, err := e.Get(ctx, key, clientv3.WithPrefix())

	if err != nil {
		logger.Error(ctx, "getInfo [%s] from etcd failed: %+v", key, err)
		return nil, err
	}

	if len(getResp.Kvs) != 0 {
		info := Info{
			Value:          string(getResp.Kvs[0].Value),
			CreateRevision: getResp.Kvs[0].CreateRevision,
			ModRevision:    getResp.Kvs[0].ModRevision,
			Version:        getResp.Kvs[0].Version,
		}
		return &info, nil
	} else {
		return nil, nil
	}
}

func putInfo(key string, info string, expireTime int64) error {
	ctx := context.Background()
	e := global.GetInstance().GetEtcd()

	resp, err := e.Grant(ctx, expireTime)
	if err != nil {
		logger.Error(ctx, "Grant TTL from etcd failed: %+v", err)
		return err
	}

	_, err = e.Put(ctx, key, info, clientv3.WithLease(resp.ID))

	if err != nil {
		logger.Error(ctx, "putInfo [%s] [%s] to etcd failed: %+v", key, info, err)
		return err
	}

	return err
}
