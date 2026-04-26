.PHONY: all build build-ro install install-ro test test-ro verify clean

GOBIN ?= $(shell go env GOPATH)/bin
PREFIX ?= $(GOBIN)

VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

all: build build-ro

build:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/freeagent ./cmd/freeagent

build-ro:
	mkdir -p bin
	go build -tags readonly $(LDFLAGS) -o bin/freeagent-ro ./cmd/freeagent

test:
	go test ./...

test-ro:
	go test -tags readonly ./...

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
