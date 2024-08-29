SHELL := /bin/bash
.SHELLFLAGS = -euo pipefail -c
.ONESHELL:
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


SEMVER_MAJOR_VERSION    ?=0
SEMVER_MINOR_VERSION    ?=0
SEMVER_PATCH_VERSION    ?=0
SEMVER_FULL_VERSION     :=$(SEMVER_MAJOR_VERSION).$(SEMVER_MINOR_VERSION).$(SEMVER_PATCH_VERSION)
SEMVER_PRERELEASE ?= dev
SEMVER_BUILDMETA  ?= 0
BUILD_DATE        :=$(shell date -u -Iseconds)
BUILD_VCS_URL     :=$(shell git config --get remote.origin.url) 
BUILD_VCS_ID      :=$(shell git log -n 1 --date=iso-strict-local --format="%h")
BUILD_VCS_ID_DATE :=$(shell TZ=UTC0 git log -n 1 --date=iso-strict-local --format='%ad')
FILE_BUILD_VERSION :=$(SEMVER_FULL_VERSION)-$(SEMVER_PRERELEASE)+$(SEMVER_BUILDMETA)

GO_LDFLAGS = -ldflags="$(BUILDFLAGS) \
	-X '$(GOMODULECMD).BuildMajorVersion=$(SEMVER_MAJOR_VERSION)' \
	-X '$(GOMODULECMD).BuildMinorVersion=$(SEMVER_MINOR_VERSION)' \
	-X '$(GOMODULECMD).BuildPatchVersion=$(SEMVER_PATCH_VERSION)' \
	-X '$(GOMODULECMD).BuildPrerelease=$(SEMVER_PRERELEASE)' \
	-X '$(GOMODULECMD).BuildMeta=$(SEMVER_BUILDMETA)' \
	-X '$(GOMODULECMD).BuildDate=$(BUILD_DATE)' \
	-X '$(GOMODULECMD).BuildVcsUrl=$(BUILD_VCS_URL)' \
	-X '$(GOMODULECMD).BuildVcsId=$(BUILD_VCS_ID)' \
	-X '$(GOMODULECMD).BuildVcsIdDate=$(BUILD_VCS_ID_DATE)'"

test_dirs=$(shell   find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)

all: test releases ## Run tests and build the binary for all platforms (Default target)

.PHONY: clean distbuild distclean linux darwin windows build fmt test check help
clean: ## Clean
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm -rf build

# Releases
releases: distclean distbuild linux darwin windows ## Build the binary for all platforms

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

build: clean ## Build the binary for your platform
	@echo "# building cli"
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(GO_LDFLAGS) -o ${BUILD_PATH}/${BUILD}-${FILE_BUILD_VERSION} .

install: build ## Install the plugin locally
	@echo "# installing plugin"
	@cf install-plugin -f ${BUILD_PATH}/${BUILD}-${FILE_BUILD_VERSION}

check: fmt lint test ## Run fmt, lint and test

fmt: ## Run goimports-reviser: Right imports sorting & code formatting tool (goimports alternative)
	@echo "# formatting code"
	@goimports-reviser -rm-unused -set-alias -format ./...

test: ## Run tests
	@echo "# running tests"
	@ginkgo -r .

lint: ## Run linter
	@echo "# running linter"
	@golangci-lint run --new-from-rev=HEAD~1

update-repo-index: # releases ## Update the repo-index.yml file
	@[ -f cli-plugin-repo/repo-index.yml ] || { echo "This target expects a checkout of https://github.com/cloudfoundry/cli-plugin-repo in the directory cli-plugin-repo"; exit 1; }
	echo "# updating repo-index.yml"
	pipenv install
	pipenv run scripts/update-cli-plugin-repo.py ${SEMVER_FULL_VERSION} osx ${BUILD}-darwin-amd64-${FILE_BUILD_VERSION}
	pipenv run scripts/update-cli-plugin-repo.py "${SEMVER_FULL_VERSION}" linux64 "${BUILD}-linux-amd64-${FILE_BUILD_VERSION}"
	pipenv run scripts/update-cli-plugin-repo.py "${SEMVER_FULL_VERSION}" win64 "${BUILD}-windows-amd64-${FILE_BUILD_VERSION}.exe"
	echo "# sorting repo-index.yml"
	pushd cli-plugin-repo
	go run sort/main.go repo-index.yml
	popd

help: ## Show this help
	@grep --extended-regexp --no-filename '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
