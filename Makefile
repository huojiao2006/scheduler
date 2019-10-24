generate: Makefile

dev:
	rm -f apiserver controller scheduler nodeagent
	echo "Building binary..."
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./apiserver cmd/apiserver/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./controller cmd/controller/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./scheduler cmd/scheduler/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags '-w' -o ./nodeagent cmd/nodeagent/main.go
	echo "Built successfully"

build:
	docker build -t scheduler -f ./Dockerfile .

clean:
	rm -f apiserver controller scheduler nodeagent


.PHONY: compose-up
compose-up: ## Launch Scheduler in docker compose
	docker-compose up -d
	@echo "compose-up done"

.PHONY: compose-down
compose-down: ## Shutdown docker compose
	docker-compose down
	@echo "compose-down done"