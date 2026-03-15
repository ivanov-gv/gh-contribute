# Load .env if it exists (gitignored, copy from .env.example)
-include .env
export

BINARY    := gh-contribute
BUILD_DIR := bin

.PHONY: build test test-integration lint fmt tidy clean install

## build: compile the binary to bin/
build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/$(BINARY)

## install: install the binary to $GOPATH/bin
install:
	go install ./cmd/$(BINARY)

## test: run unit tests with race detector
test:
	go test -count=1 -race ./internal/...

## test-integration: run integration tests with race detector
test-integration:
	go test -count=1 -race ./test/...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format all Go source files
fmt:
	gofmt -w ./...

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
