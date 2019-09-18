package apiserver

import (
	//"fmt"

	"context"
	"encoding/json"
	"strings"

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
	Key            string `json:"Key"`
	Value          []byte `json:"Value"`
	CreateRevision int64  `json:"CreateRevision"`
	ModRevision    int64  `json:"ModRevision"`
	Version        int64  `json:"Version"`
}

type Event struct {
	Event string `json:"Event"`
	Data  Info   `json:"Data"`
}

type Watcher struct {
	key       string
	filter    string
	storage   map[string][]Event
	eventChan chan Event
	stopChan  chan string
}

func NewWatcher(key string, initValue []Info, filter string) *Watcher {
	storage := make(map[string][]Event)

	if initValue != nil {
		for _, info := range initValue {
			storage[info.Key] = []Event{}
			storage[info.Key] = append(storage[info.Key], Event{"ADD", info})
		}
	}

	return &Watcher{
		key:       key,
		filter:    filter,
		storage:   storage,
		eventChan: make(chan Event, 10),
		stopChan:  make(chan string, 1),
	}
}

func filterEvent(value []byte, filter string) bool {
	logger.Info(nil, "filterEvent [%v] [%s]", value, filter)

	var map_value map[string]interface{}

	err := json.Unmarshal(value, &map_value)
	if err != nil {
		logger.Error(nil, "filterEvent error [%v]", err)
		return false
	}

	if filter == "" {
		return true
	}

	expr := strings.Split(filter, "=")
	left := ""
	right := ""
	if len(expr) == 2 {
		left = expr[0]
		right = expr[1]
	} else if len(expr) == 1 {
		left = expr[0]
	} else {
		return true
	}

	if val, ok := map_value[left]; ok {
		if right == val.(string) {
			return true
		} else {
			return false
		}
	} else {
		return false
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

	for _, v := range wc.storage {
		if len(v) > 0 {
			if filterEvent(v[0].Data.Value, wc.filter) {
				wc.eventChan <- v[0]
			} else {
				v[0].Event = "DELETE"
			}
		}
	}

	for res := range watchRes {
		for _, ev := range res.Events {
			if ev.Type == mvccpb.PUT {
				key := string(ev.Kv.Key)
				logger.Info(nil, "watch got put event [%s] [%s]", key, string(ev.Kv.Value))
				info := Info{
					Key:            key,
					Value:          ev.Kv.Value,
					CreateRevision: ev.Kv.CreateRevision,
					ModRevision:    ev.Kv.ModRevision,
					Version:        ev.Kv.Version,
				}

				notifyWatcher := true
				event := "MODIFY"

				var eventStore []Event
				if v, ok := wc.storage[key]; ok {
					eventStore = v
				} else {
					wc.storage[key] = []Event{}
					eventStore = wc.storage[key]
				}

				if filterEvent(info.Value, wc.filter) {
					if len(eventStore) == 0 {
						event = "ADD"
					} else if eventStore[len(eventStore)-1].Event == "DELETE" {
						event = "ADD"
					}
				} else {
					if len(eventStore) == 0 {
						notifyWatcher = false
					} else if eventStore[len(eventStore)-1].Event == "DELETE" {
						notifyWatcher = false
					}
					event = "DELETE"
				}

				eventNotify := Event{
					Event: event,
					Data:  info,
				}
				if notifyWatcher {
					wc.eventChan <- eventNotify
				}

				wc.storage[key] = append(wc.storage[key], eventNotify)
			} else if ev.Type == mvccpb.DELETE {
				key := string(ev.Kv.Key)
				logger.Info(nil, "watch got delete event [%s] [%s]", key, string(ev.Kv.Value))
				info := Info{
					Key:            key,
					Value:          ev.Kv.Value,
					CreateRevision: ev.Kv.CreateRevision,
					ModRevision:    ev.Kv.ModRevision,
					Version:        ev.Kv.Version,
				}

				notifyWatcher := true

				var eventStore []Event
				if v, ok := wc.storage[key]; ok {
					eventStore = v
				} else {
					wc.storage[key] = []Event{}
					eventStore = wc.storage[key]
				}

				if len(eventStore) == 0 {
					notifyWatcher = false
				} else if eventStore[len(eventStore)-1].Event == "DELETE" {
					notifyWatcher = false
				}

				eventNotify := Event{
					Event: "DELETE",
					Data:  info,
				}
				if notifyWatcher {
					wc.eventChan <- eventNotify
				}

				wc.storage[key] = append(wc.storage[key], eventNotify)
			}
		}
	}

	logger.Info(nil, "watch ended")
}

func (wc *Watcher) stop() {
	wc.stopChan <- "close"
}

func listWatch(fnName string, key string, filter string, watch bool, response *restful.Response) {
	initValue, err := getInfo(key)
	if err != nil {
		logger.Debug(nil, "%s request data error %+v.", fnName, err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	logger.Debug(nil, "%s success", fnName)

	if watch {
		watcher := NewWatcher(key, initValue, filter)
		go watcher.watch()

		notify := response.CloseNotify()

		for {
			select {
			case event := <-watcher.eventChan:
				response.Write([]byte(formatEvent(event) + "\n"))
				response.Flush()
				logger.Info(nil, "%s got event [%v]", fnName, event)
			case <-notify:
				watcher.stop()
				logger.Info(nil, "%s disconnected", fnName)
				return
			}
		}
	} else {
		for _, info := range initValue {
			if filterEvent(info.Value, filter) {
				response.Write([]byte(formatInfo(&info) + "\n"))
			}
		}
	}
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
	watch := parseBool(request.QueryParameter("watch"))
	filter := request.QueryParameter("filter")

	key := "nodes/"

	listWatch("DescribeNodes", key, filter, watch, response)
}

func DescribeNode(request *restful.Request, response *restful.Response) {
	node := request.PathParameter("node_name")
	watch := parseBool(request.QueryParameter("watch"))
	filter := request.QueryParameter("filter")

	key := "nodes/" + node

	listWatch("DescribeNodes", key, filter, watch, response)
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

func getInfo(key string) ([]Info, error) {
	ctx := context.Background()
	e := global.GetInstance().GetEtcd()

	getResp, err := e.Get(ctx, key, clientv3.WithPrefix())

	if err != nil {
		logger.Error(ctx, "getInfo [%s] from etcd failed: %+v", key, err)
		return nil, err
	}

	if len(getResp.Kvs) != 0 {
		var infos []Info
		for _, kv := range getResp.Kvs {
			info := Info{
				Key:            string(kv.Key),
				Value:          kv.Value,
				CreateRevision: kv.CreateRevision,
				ModRevision:    kv.ModRevision,
				Version:        kv.Version,
			}

			infos = append(infos, info)
		}

		return infos, nil
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
