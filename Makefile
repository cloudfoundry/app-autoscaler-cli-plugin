SHELL := /bin/bash
.SHELLFLAGS = -euo pipefail -c
CGO_ENABLED = 0
BUILDTAGS :=
BUILD_PATH:=build
BUILD:=ascli
ifeq ($(CGO_ENABLED),1)
BUILDFLAGS += -linkmode=external
else
BUILDFLAGS :=
endif
GOOS           :=$(shell go env GOOS)
GOARCH         :=$(shell go env GOARCH)
GOMODULECMD    :=main


SEMVER_VERSION    ?=3.0.0
SEMVER_PRERELEASE ?=
SEMVER_BUILDMETA  ?= +3
BUILD_DATE        :=$(shell date -u -Iseconds)
BUILD_VCS_URL     :=$(shell git config --get remote.origin.url) 
BUILD_VCS_ID      :=$(shell git log -n 1 --date=iso-strict-local --format="%h")
BUILD_VCS_ID_DATE :=$(shell TZ=UTC0 git log -n 1 --date=iso-strict-local --format='%ad')
FILE_BUILD_VERSION :=$(SEMVER_VERSION)$(SEMVER_PRERELEASE)$(SEMVER_BUILDMETA)

GO_LDFLAGS = -ldflags="$(BUILDFLAGS) \
			    -X '$(GOMODULECMD).BuildVersion=$(SEMVER_VERSION)' \
	            -X '$(GOMODULECMD).BuildPrerelease=$(SEMVER_PRERELEASE)' \
	            -X '$(GOMODULECMD).BuildMeta=$(SEMVER_BUILDMETA)' \
	            -X '$(GOMODULECMD).BuildDate=$(BUILD_DATE)' \
	            -X '$(GOMODULECMD).BuildVcsUrl=$(BUILD_VCS_URL)' \
	            -X '$(GOMODULECMD).BuildVcsId=$(BUILD_VCS_ID)' \
		    -X '$(GOMODULECMD).BuildVcsIdDate=$(BUILD_VCS_ID_DATE)'"

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
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-linux-amd64-${FILE_BUILD_VERSION} .
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-linux-arm64-${FILE_BUILD_VERSION} .

darwin:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-darwin-amd64-${FILE_BUILD_VERSION} .
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-darwin-arm64-${FILE_BUILD_VERSION} .
	
windows:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build ${GO_LDFLAGS} -o ${BUILD_PATH}/${BUILD}-windows-amd64-${FILE_BUILD_VERSION}.exe .

build:
	@echo "# building cli"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(GO_LDFLAGS) -o ${BUILD_PATH}/${BUILD}-${FILE_BUILD_VERSION} .


test:
	@ginkgo .

