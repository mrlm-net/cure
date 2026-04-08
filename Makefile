.PHONY: build test bench lint clean docker-build docker-push gui-deps gui-frontend gui-build gui-dev

BINARY    := cure
MODULE    := github.com/mrlm-net/cure
BUILD_DIR := bin
IMAGE     ?= ghcr.io/mrlm-net/cure
TAG       ?= latest

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cure

test:
	go test -tags no_frontend -race -count=1 ./...

bench:
	go test -bench=. -benchmem -benchtime=3s ./pkg/fs/...

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

## Frontend targets
gui-deps:
	cd frontend && npm ci

gui-frontend: gui-deps
	cd frontend && npm run build

gui-build: gui-frontend
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cure

gui-dev:
	cd frontend && npm run dev
