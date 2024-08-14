SHELL := /bin/bash
.SHELLFLAGS = -euo pipefail -c
# MAKEFLAGS = -s
CGO_ENABLED = 0
BUILDTAGS :=
BUILD_PATH:=build
BUILD:=ascli
ifeq ($(CGO_ENABLED),1)
BUILDFLAGS := -ldflags '-linkmode=external'
else
BUILDFLAGS :=
endif
GO_LDFLAGS := ${BUILDFLAGS}
test_dirs=$(shell   find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)

all: releases

.PHONY: clean distbuild distclean linux darwin windows
clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm -rf build

# Releases
releases: distclean distbuild linux darwin # windows

distbuild:
	mkdir -p ${BUILD_PATH}

distclean:
	rm -rf ${BUILD_PATH}

linux:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-linux-amd64 .
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-linux-arm64 .

darwin:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-darwin-amd64 .
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-darwin-arm64 .
	
windows:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-windows-amd64.exe .

build:
	@echo "# building cli"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* main.go


test:
	@ginkgo .

