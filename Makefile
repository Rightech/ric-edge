SHELL := /bin/bash

PROJECT_NAME ?= ric-edge

help:
	@grep -h "##" $(MAKEFILE_LIST) | \
		grep -v grep | sed -e 's/\\$$//' | sed -e 's/##//'

-include .env
export

# ensure that vendoring is enabled
_ensure_vendor:
ifeq ($(findstring -mod,$(GOFLAGS)),)
	$(eval export GOFLAGS=$(GOFLAGS) -mod=vendor)
endif

-include _ensure_vendor

define _list
	$(shell go list ./... | grep -Ev '/cmd|/third_party|/test|/tools|/api')
endef

test:	##
	## run tests and creates coverage report (see coverage.html)
	## you can add LONG=1 to run long tests too
	## also you can run concreet tests by set NAME="<test_name>"
	go test $(if $(LONG),-timeout 100m,-short) -race -cover \
		-failfast -coverprofile=coverage.out -v $(call _list) \
		$(if $(NAME),-run "^($(NAME))$$")
	@go tool cover -func coverage.out | grep "total" | xargs
	@go tool cover -html=coverage.out -o coverage.html

bench:  ##
	## run benchmark tests
	go test -benchmem -v $(call _list) \
		-bench $(if $(NAME),"^($(NAME))$$",.) \
		-run=^Benchmark

lint:  ##
	## lint project by https://github.com/golangci/golangci-lint
	@golangci-lint run --enable-all

# "|| true" required for prevent make to output error message when app
# stopped by Ctrl+C
HS ?= 1
run_%:	##
	## run service specified by %
	## you can set HS to increase data racer history size
	## (see https://golang.org/doc/articles/race_detector.html#Options)
	## also you can disable data racer via NORACE=1
	GORACE="history_size=$(HS)" go run \
		$(if $(NORACE),,-race )./cmd/$(subst _,-,$*) $(FLAGS) || true

tool_%:  ##
	## run any tool from ./tools folder
	## you can use FLAGS var to add flags for tool
	## usage:
	## for example you have ./tools/test-call tool
	## so you can run it by
	## make run_test_call or make run_test-call
	go run ./tools/$(subst _,-,$*) $(FLAGS)

VERSION := $(if $(VERSION),$(VERSION),\
	$(shell git describe --tags --always | tail -c +2))

build_%:  ##
	## build any service specified by %
	@./scripts/build.sh ./cmd/$(subst _,-,$*) $(VERSION)

buildall: build_core build_modbus build_opcua build_snmp

gen:  ##
	## run go generate
	go generate ./...

validate:  ##
	## run some project validation scripts
	@scripts/validate-license.sh

define compose
	docker-compose -p $(PROJECT_NAME) -f tools/dev-env/docker-compose.yml $(1)
endef

dev_env:  ##
	## setup dev env
	$(call compose,pull)
	$(call compose,up -d)

stop_env:  ##
	## stop dev environment
	$(call compose,stop)

rm_env:  ##
	## remove dev environment
	$(call compose,rm)
