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

	"openpitrix.io/scheduler/pkg/constants"
	"openpitrix.io/scheduler/pkg/global"
	"openpitrix.io/scheduler/pkg/logger"
	"openpitrix.io/scheduler/pkg/models"
)

func parseBool(input string) bool {
	if input == "true" {
		return true
	} else {
		return false
	}
}

type Error struct {
	Message string `json:"message" description:"error message"`
}

func Wrap(err error) Error {
	return Error{Message: err.Error()}
}

type Watcher struct {
	key       string
	filter    string
	storage   map[string][]models.Event
	eventChan chan models.Event
	stopChan  chan string
}

func NewWatcher(key string, initValue []models.Info, filter string) *Watcher {
	storage := make(map[string][]models.Event)

	if initValue != nil {
		for _, info := range initValue {
			storage[info.Key] = []models.Event{}
			storage[info.Key] = append(storage[info.Key], models.Event{"ADD", info})
		}
	}

	return &Watcher{
		key:       key,
		filter:    filter,
		storage:   storage,
		eventChan: make(chan models.Event, 10),
		stopChan:  make(chan string, 1),
	}
}

func evaluateRule(rule string, map_value map[string]interface{}) bool {
	expr := strings.Split(rule, "=")
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

func filterEvent(value []byte, filter string) bool {
	logger.Info(nil, "filterEvent [%s] [%s]", value, filter)

	if len(value) == 0 {
		return true
	}

	var map_value map[string]interface{}

	err := json.Unmarshal(value, &map_value)
	if err != nil {
		logger.Error(nil, "filterEvent error [%v]", err)
		return false
	}

	if filter == "" {
		return true
	}

	result := true

	rules := strings.Split(filter, ",")

	for _, rule := range rules {
		if !evaluateRule(rule, map_value) {
			result = false
			break
		}
	}

	return result
}

func (wc *Watcher) watch() {
	e := global.GetInstance().GetEtcd()

	defer close(wc.eventChan)
	defer close(wc.stopChan)

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
				info := models.Info{
					Key:            key,
					Value:          ev.Kv.Value,
					CreateRevision: ev.Kv.CreateRevision,
					ModRevision:    ev.Kv.ModRevision,
					Version:        ev.Kv.Version,
				}

				notifyWatcher := true
				event := "MODIFY"

				var eventStore []models.Event
				if v, ok := wc.storage[key]; ok {
					eventStore = v
				} else {
					wc.storage[key] = []models.Event{}
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

				eventNotify := models.Event{
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
				info := models.Info{
					Key:            key,
					Value:          ev.Kv.Value,
					CreateRevision: ev.Kv.CreateRevision,
					ModRevision:    ev.Kv.ModRevision,
					Version:        ev.Kv.Version,
				}

				notifyWatcher := true

				var eventStore []models.Event
				if v, ok := wc.storage[key]; ok {
					eventStore = v
				} else {
					wc.storage[key] = []models.Event{}
					eventStore = wc.storage[key]
				}

				if len(eventStore) == 0 {
					notifyWatcher = false
				} else if eventStore[len(eventStore)-1].Event == "DELETE" {
					notifyWatcher = false
				}

				eventNotify := models.Event{
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

func formatEvent(event models.Event) string {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Error(nil, "formatEvent error [%v]", err)
	}

	return string(eventBytes)
}

func formatInfo(info *models.Info) string {
	if nil == info {
		return "{}"
	}

	infoBytes, _ := json.Marshal(*info)

	return string(infoBytes)
}

func getInfo(key string) ([]models.Info, error) {
	ctx := context.Background()
	e := global.GetInstance().GetEtcd()

	getResp, err := e.Get(ctx, key, clientv3.WithPrefix())

	if err != nil {
		logger.Error(ctx, "getInfo [%s] from etcd failed: %+v", key, err)
		return nil, err
	}

	if len(getResp.Kvs) != 0 {
		var infos []models.Info
		for _, kv := range getResp.Kvs {
			info := models.Info{
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

	if expireTime <= 0 {
		_, err := e.Put(ctx, key, info)

		if err != nil {
			logger.Error(ctx, "putInfo [%s] [%s] to etcd failed: %+v", key, info, err)
			return err
		}
		return err
	} else {
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
}

func CreateNode(request *restful.Request, response *restful.Response) {
	node := request.PathParameter("node_name")
	nodeInfo := new(models.APIInfo)

	err := request.ReadEntity(&nodeInfo)
	if err != nil {
		logger.Error(nil, "CreateNode request data error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	ttlValue := nodeInfo.TTL

	if ttlValue > constants.TTLMax {
		ttlValue = constants.TTLMax
	}

	if ttlValue < constants.TTLMin {
		ttlValue = constants.TTLMin
	}

	key := "nodes/" + node

	err = putInfo(key, nodeInfo.Info, ttlValue)
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

func CreateTask(request *restful.Request, response *restful.Response) {
	task := request.PathParameter("task_name")
	taskInfo := new(models.APIInfo)

	err := request.ReadEntity(&taskInfo)
	if err != nil {
		logger.Error(nil, "CreateTask request data error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	key := "tasks/" + task

	err = putInfo(key, taskInfo.Info, -1)
	if err != nil {
		logger.Debug(nil, "CreateTask putInfo error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	logger.Debug(nil, "CreateTask success")

	response.WriteHeaderAndEntity(http.StatusOK, "task")
}

func DescribeTasks(request *restful.Request, response *restful.Response) {
	watch := parseBool(request.QueryParameter("watch"))
	filter := request.QueryParameter("filter")

	key := "tasks/"

	listWatch("DescribeTasks", key, filter, watch, response)
}

func CreateJob(request *restful.Request, response *restful.Response) {
	job := request.PathParameter("job_name")
	jobInfo := new(models.APIInfo)

	err := request.ReadEntity(&jobInfo)
	if err != nil {
		logger.Error(nil, "CreateJob request data error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	key := "jobs/" + job

	err = putInfo(key, jobInfo.Info, -1)
	if err != nil {
		logger.Debug(nil, "CreateJob putInfo error %+v.", err)
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Wrap(err))
		return
	}

	logger.Debug(nil, "CreateJob success")

	response.WriteHeaderAndEntity(http.StatusOK, "job")
}

func DescribeJobs(request *restful.Request, response *restful.Response) {
	watch := parseBool(request.QueryParameter("watch"))
	filter := request.QueryParameter("filter")

	key := "jobs/"

	listWatch("DescribeJobs", key, filter, watch, response)
}
