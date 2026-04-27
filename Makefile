.PHONY: all build build-ro install install-ro test test-ro test-e2e test-e2e-ro verify clean

GOBIN ?= $(shell go env GOPATH)/bin
PREFIX ?= $(GOBIN)

all: build build-ro

build:
	mkdir -p bin
	go build -o bin/freeagent ./cmd/freeagent

build-ro:
	mkdir -p bin
	go build -tags readonly -o bin/freeagent-ro ./cmd/freeagent

test:
	go test ./...

test-ro:
	go test -tags readonly ./...

test-e2e: build
	go test -tags e2e -count=1 -timeout 15m -parallel 1 ./internal/e2e/...

test-e2e-ro: build-ro
	go test -tags e2e,readonly -count=1 -timeout 15m -parallel 1 ./internal/e2e/...

install-ro: test-ro build-ro
	install -m 0755 bin/freeagent-ro "$(PREFIX)/freeagent-ro"
	@echo "installed $(PREFIX)/freeagent-ro"

install: test build
	install -m 0755 bin/freeagent "$(PREFIX)/freeagent"
	@echo "installed $(PREFIX)/freeagent"

verify:
	go test ./...
	go test -tags readonly ./...
	go vet ./...

clean:
	rm -rf bin/
