# Scheduler
[![License](http://img.shields.io/badge/license-apache%20v2-blue.svg)](https://github.com/KubeSphere/KubeSphere/blob/master/LICENSE)

----

## Introduction

Scheduler is a general purpose scheduler for cluster job/crobjob scheduling.

## Usage

观察task情况
```
curl "http://127.0.0.1:8080/api/v1alpha1/tasks/?watch=true"
```

观察job情况
```
curl "http://127.0.0.1:8080/api/v1alpha1/jobs/?watch=true"
```

观察cron情况
```
curl "http://127.0.0.1:8080/api/v1alpha1/crons/?watch=true"
```

创建两个cron
```
curl -H "Accept: application/json" -H "Content-type: application/json" -X POST -d '{"Info": "{\"Name\":\"c-1234abcd\",\"Script\":\"* * * * *\",\"Cmd\":[\"curl\"]}"}' http://127.0.0.1:8080/api/v1alpha1/crons/c-1234abcd
curl -H "Accept: application/json" -H "Content-type: application/json" -X POST -d '{"Info": "{\"Name\":\"c-1234defg\",\"Script\":\"*/2 * * * *\"}"}' http://127.0.0.1:8080/api/v1alpha1/crons/c-1234defg
```

删除cron
```
curl -XDELETE http://127.0.0.1:8080/api/v1alpha1/crons/c-1234abcd
```

查看etcd信息

节点
```
ETCDCTL_API=3 etcdctl get --prefix scheduler/nodes -w=json | jq .
```

tasks
```
ETCDCTL_API=3 etcdctl get --prefix scheduler/tasks -w=json | jq .
```

jobs
```
ETCDCTL_API=3 etcdctl get --prefix scheduler/jobs -w=json | jq .
```

crons
```
ETCDCTL_API=3 etcdctl get --prefix scheduler/crons -w=json | jq .
```

清除etcd
```
ETCDCTL_API=3 etcdctl del --prefix=true scheduler/tasks -w=json | jq .
ETCDCTL_API=3 etcdctl del --prefix=true scheduler/jobs -w=json | jq .
ETCDCTL_API=3 etcdctl del --prefix=true scheduler/crons -w=json | jq .
```
