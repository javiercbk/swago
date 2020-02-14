GIT_DATE := $(firstword $(shell git --no-pager show --date=short --format="%ai" --name-only))
GIT_VERSION := $(shell git rev-parse HEAD)
BIN_VERSION := $(GIT_VERSION)|$(GIT_DATE)
MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CUR_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

.PHONY: swago

swago:
	go build cmd/swago/swago.go

# remove unused dependencies and tidy up modules
mod-tydy:
	go mod tidy

# lints the project
lint:
	$(GOPATH)/bin/golangci-lint run

# outputs the current version
version:
	@echo "$(BIN_VERSION)"

