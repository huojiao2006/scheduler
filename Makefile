generate: Makefile

dev:
	rm -f apiserver controller scheduler nodeagent
	echo "Building binary..."
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./apiserver cmd/apiserver/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./controller cmd/controller/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./scheduler cmd/scheduler/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./nodeagent cmd/nodeagent/main.go
	echo "Built successfully"

clean:
	rm -f apiserver controller scheduler nodeagent