SHELL := /bin/bash
.SHELLFLAGS = -euo pipefail -c
MAKEFLAGS = -s
CGO_ENABLED = 1
BUILDTAGS :=
BUILDFLAGS := -ldflags '-linkmode=external'

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm -rf build

build:
	@echo "# building cli"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* src/cli/main.go
