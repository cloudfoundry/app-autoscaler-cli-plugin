SHELL := /bin/bash
.SHELLFLAGS = -euo pipefail -c
MAKEFLAGS = -s
CGO_ENABLED = 1
BUILDTAGS :=
BUILDFLAGS := -ldflags '-linkmode=external'
test_dirs=$(shell   find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm -rf build

build:
	@echo "# building cli"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* main.go

build_tests: $(addprefix build_test-,$(test_dirs))

build_test-%:
	@echo " - building '$*' tests"
	@export build_folder=${PWD}/build/tests/$* &&\
	 mkdir -p $${build_folder} &&\
	 cd $* &&\
	 for package in $$(  go list ./... | sed 's|.*/autoscaler/$*|.|' | awk '{ print length, $$0 }' | sort -n -r | cut -d" " -f2- );\
	 do\
	   export test_file=$${build_folder}/$${package}.test;\
	   echo "   - compiling $${package} to $${test_file}";\
	   go test -c -o $${test_file} $${package};\
	 done;
