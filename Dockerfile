FROM golang:1.12 as golang

ADD . /go/src/openpitrix.io/scheduler
WORKDIR /go/src/openpitrix.io/scheduler

RUN mkdir -p /scheduler_bin
RUN CGO_ENABLED=0 GO111MODULE=on go build -v -a -installsuffix cgo -ldflags '-w' -o /scheduler_bin/apiserver cmd/apiserver/main.go
RUN CGO_ENABLED=0 GO111MODULE=on go build -v -a -installsuffix cgo -ldflags '-w' -o /scheduler_bin/nodeagent cmd/nodeagent/main.go
RUN CGO_ENABLED=0 GO111MODULE=on go build -v -a -installsuffix cgo -ldflags '-w' -o /scheduler_bin/controller cmd/controller/main.go
RUN CGO_ENABLED=0 GO111MODULE=on go build -v -a -installsuffix cgo -ldflags '-w' -o /scheduler_bin/scheduler cmd/scheduler/main.go

FROM alpine:3.9
COPY --from=golang /scheduler_bin/apiserver /scheduler/apiserver
COPY --from=golang /scheduler_bin/nodeagent /scheduler/nodeagent
COPY --from=golang /scheduler_bin/controller /scheduler/controller
COPY --from=golang /scheduler_bin/scheduler /scheduler/scheduler

EXPOSE 8080
EXPOSE 8081