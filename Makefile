MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
LOCAL_BIN_PATH := ${PROJECT_PATH}/bin

LINT_GOGC := 10
LINT_TIMEOUT := 10m

## Tools
GOLANGCI ?= $(LOCALBIN)/golangci-lint
GOLANGCI_VERSION ?= v2.1.6
KUBECTL ?= kubectl

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


ifndef ignore-not-found
  ignore-not-found = false
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: test

.PHONY: clean
clean:
	go clean -x
	go clean -x -testcache

.PHONY: fmt
fmt: golangci-lint
	@$(GOLANGCI) fmt --config .golangci.yml
	go fmt ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: deps
deps:
	go mod tidy

.PHONY: check/lint
check: check/lint

.PHONY: check/lint
check/lint: golangci-lint
	@$(GOLANGCI) run --config .golangci.yml --timeout $(LINT_TIMEOUT)

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI)
$(GOLANGCI): $(LOCALBIN)
	@test -s $(GOLANGCI) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

