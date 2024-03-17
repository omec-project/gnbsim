# SPDX-License-Identifier: Apache-2.0
# Copyright 2021 Open Networking Foundation
# Copyright 2022-present Intel Corporation
#

PROJECT_NAME             := gnbSim
DOCKER_VERSION           ?= $(shell cat ./VERSION)

## Docker related
DOCKER_REGISTRY          ?=
DOCKER_REPOSITORY        ?=
DOCKER_TAG               ?= ${DOCKER_VERSION}
DOCKER_IMAGENAME         := ${DOCKER_REGISTRY}${DOCKER_REPOSITORY}${PROJECT_NAME}:${DOCKER_TAG}
DOCKER_BUILDKIT          ?= 1
DOCKER_BUILD_ARGS        ?=

## Docker labels. Only set ref and commit date if committed
DOCKER_LABEL_VCS_URL     ?= $(shell git remote get-url $(shell git remote))
DOCKER_LABEL_VCS_REF     ?= $(shell git diff-index --quiet HEAD -- && git rev-parse HEAD || echo "unknown")
DOCKER_LABEL_COMMIT_DATE ?= $(shell git diff-index --quiet HEAD -- && git show -s --format=%cd --date=iso-strict HEAD || echo "unknown" )
DOCKER_LABEL_BUILD_DATE  ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

DOCKER_TARGETS           ?= gnbsim

GO_BIN_PATH = bin
GO_SRC_PATH = ./
C_BUILD_PATH = build
ROOT_PATH = $(shell pwd)

NF = $(GO_NF)
GO_NF = gnbsim

NF_GO_FILES = $(shell find $(GO_SRC_PATH)/$(%) -name "*.go" ! -name "*_test.go")

VERSION = $(shell git describe --tags)
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH = $(shell git submodule status | grep $(GO_SRC_PATH)/$(@F) | awk '{print $$(1)}' | cut -c1-8)
COMMIT_TIME = $(shell cd $(GO_SRC_PATH) && git log --pretty="%ai" -1 | awk '{time=$$(1)"T"$$(2)"Z"; print time}')

.PHONY: $(NF) clean docker-build docker-push

.DEFAULT_GOAL: nfs

nfs: $(NF)

all: $(NF)

$(GO_NF): % : $(GO_BIN_PATH)/%

$(GO_BIN_PATH)/%: %.go $(NF_GO_FILES)
# $(@F): The file-within-directory part of the file name of the target.
	@echo "Start building $(@F)...."
	cd $(GO_SRC_PATH)/ && \
	CGO_ENABLED=0 go build -o $(ROOT_PATH)/$@ $(@F).go

vpath %.go $(addprefix $(GO_SRC_PATH)/, $(GO_NF))

clean:
	rm -rf $(addprefix $(GO_BIN_PATH)/, $(GO_NF))
	rm -rf $(addprefix $(GO_SRC_PATH)/, $(addsuffix /$(C_BUILD_PATH), $(C_NF)))

docker-build:
	@go mod vendor
	for target in $(DOCKER_TARGETS); do \
		DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build  $(DOCKER_BUILD_ARGS) \
			--target $$target \
			--tag ${DOCKER_REGISTRY}${DOCKER_REPOSITORY}5gc-$$target:${DOCKER_TAG} \
			--build-arg org_label_schema_version="${DOCKER_VERSION}" \
			--build-arg org_label_schema_vcs_url="${DOCKER_LABEL_VCS_URL}" \
			--build-arg org_label_schema_vcs_ref="${DOCKER_LABEL_VCS_REF}" \
			--build-arg org_label_schema_build_date="${DOCKER_LABEL_BUILD_DATE}" \
			--build-arg org_opencord_vcs_commit_date="${DOCKER_LABEL_COMMIT_DATE}" \
			. \
			|| exit 1; \
	done
	rm -rf vendor

docker-push:
	for target in $(DOCKER_TARGETS); do \
		docker push ${DOCKER_REGISTRY}${DOCKER_REPOSITORY}5gc-$$target:${DOCKER_TAG}; \
	done

.coverage:
	rm -rf $(CURDIR)/.coverage
	mkdir -p $(CURDIR)/.coverage

test: .coverage
	docker run --rm -v $(CURDIR):/gnbsim -w /gnbsim golang:latest \
		go test \
			-race \
			-failfast \
			-coverprofile=.coverage/coverage-unit.txt \
			-covermode=atomic \
			-v \
			./ ./...


fmt:
	@go fmt ./...

golint:
	@docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:latest golangci-lint run -v --config /app/.golangci.yml

check-reuse:
	@docker run --rm -v $(CURDIR):/gnbsim -w /gnbsim omecproject/reuse-verify:latest reuse lint
