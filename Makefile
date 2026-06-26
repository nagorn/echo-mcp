APP := echo-mcp
IMAGE := echo-mcp:local
GO_CACHE := /private/tmp/echo-mcp-go-cache

.PHONY: test vet build run docker-build

test:
	GOCACHE=$(GO_CACHE) go test ./...

vet:
	GOCACHE=$(GO_CACHE) go vet ./...

build:
	GOCACHE=$(GO_CACHE) go build -o bin/$(APP) ./cmd/echo-mcp

run:
	GOCACHE=$(GO_CACHE) go run ./cmd/echo-mcp

docker-build:
	docker build -t $(IMAGE) .
