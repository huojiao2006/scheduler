package apiserver

import (
	//"fmt"

	"context"

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

type Event struct {
	event string
	value string
}

type Watcher struct {
	key       string
	eventChan chan Event
	stopChan  chan string
}

func NewWatcher(key string) *Watcher {
	return &Watcher{
		key:       key,
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

	for res := range watchRes {
		for _, ev := range res.Events {
			if ev.Type == mvccpb.PUT {
				logger.Info(nil, "watch got put event [%s] [%s]", string(ev.Kv.Key), string(ev.Kv.Value))
				wc.eventChan <- Event{"put", string(ev.Kv.Value)}
			} else if ev.Type == mvccpb.DELETE {
				logger.Info(nil, "watch got delete event [%s] [%s]", string(ev.Kv.Key), string(ev.Kv.Value))
				wc.eventChan <- Event{"delete", string(ev.Kv.Value)}
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

	result, err := getInfo(key)
	if err != nil {
		logger.Debug(nil, "CreateNode request data error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	logger.Debug(nil, "DescribeNodes success")

	if watch {
		watcher := NewWatcher(key)
		go watcher.watch()

		response.Write([]byte(result + "\n"))
		response.Flush()

		notify := response.CloseNotify()

		for {
			select {
			case event := <-watcher.eventChan:
				response.Write([]byte(event.event + " " + event.value + "\n"))
				response.Flush()
				logger.Info(nil, "DescribeNodes got event [%s] [%s]", event.event, event.value)
			case <-notify:
				watcher.stop()
				logger.Info(nil, "DescribeNodes disconnected")
				return
			}
		}
	} else {
		response.Write([]byte(result + "\n"))
	}
}

func getInfo(key string) (string, error) {
	ctx := context.Background()
	e := global.GetInstance().GetEtcd()

	getResp, err := e.Get(ctx, key, clientv3.WithPrefix())

	if err != nil {
		logger.Error(ctx, "getInfo [%s] from etcd failed: %+v", key, err)
		return "", err
	}

	result := ""

	if len(getResp.Kvs) != 0 {
		result = string(getResp.Kvs[0].Value)
	}

	return result, nil
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
