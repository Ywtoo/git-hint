SHELL := /bin/sh

BIN := bin/githint
GOCACHE_DIR := /tmp/git-hint-gocache
VERSION ?= dev

.PHONY: help build test rebuild format lint check install-local update-local uninstall-local clean

help:
	@echo "make build                         Compile bin/githint"
	@echo "make test                          Run all Go tests"
	@echo "make rebuild                       Rebuild the local command index"
	@echo "make install-local VERSION=x.y.z   Install locally"
	@echo "make update-local VERSION=x.y.z    Update local installation"
	@echo "make uninstall-local               Remove local installation"

build:
	@mkdir -p bin
	GOCACHE=$(GOCACHE_DIR) go build -ldflags "-X git-hint/internal/version.Value=$(VERSION)" -o $(BIN) .
	@echo "built $(BIN)"

test:
	GOCACHE=$(GOCACHE_DIR) go test ./...

rebuild: build
	$(BIN) rebuild

install-local:
	VERSION=$(VERSION) ./tools/install/local-install.sh install

update-local:
	VERSION=$(VERSION) ./tools/install/local-install.sh update

uninstall-local:
	./tools/install/local-install.sh uninstall

format:
	goimports -w .

lint:
	golangci-lint run --fix ./...

check: format
	GOCACHE=$(GOCACHE_DIR) go test ./...

clean:
	rm -rf bin
