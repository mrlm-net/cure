---
title: "Installation"
description: "Install the cure CLI via go install or build from source"
order: 1
section: "getting-started"
---

# Installation

## Requirements

- Go 1.25 or later

## Install via go install

Install the latest stable release using `go install`:

```sh
go install github.com/mrlm-net/cure/cmd/cure@latest
```

This downloads, compiles, and installs the `cure` binary to your `$GOPATH/bin` (usually `~/go/bin`). Make sure `$GOPATH/bin` is on your `PATH`.

## Verify installation

```sh
cure version
```

You should see version and build information printed to stdout.

## Build from source

To build from source, clone the repository and use the provided Makefile:

```sh
git clone https://github.com/mrlm-net/cure.git
cd cure
make build
./bin/cure version
```

The compiled binary is placed in `bin/cure`. You can move it to any directory on your `PATH`:

```sh
sudo mv bin/cure /usr/local/bin/cure
```

## Build commands

| Command | Purpose |
|---------|---------|
| `make build` | Build the `cure` binary to `bin/` |
| `make test` | Run all tests with race detector |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run `go vet` for static analysis |
| `make clean` | Remove build artifacts |
