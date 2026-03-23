.PHONY: build test lint clean docker-build docker-push

BINARY    := cure
MODULE    := github.com/mrlm-net/cure
BUILD_DIR := bin
IMAGE     ?= ghcr.io/mrlm-net/cure
TAG       ?= latest

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cure

test:
	go test -race -count=1 ./...

test-coverage:
	go test -race -coverprofile=coverage.out -coverpkg=./pkg/agent/... ./...
	go tool cover -func=coverage.out

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out

docker-build:
	docker build --tag $(IMAGE):$(TAG) .

docker-push:
	docker push $(IMAGE):$(TAG)
