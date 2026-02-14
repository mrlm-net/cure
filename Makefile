.PHONY: build test lint clean

BINARY := cure
MODULE := github.com/mrlm-net/cure
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cure

test:
	go test -race -count=1 ./...

test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out
