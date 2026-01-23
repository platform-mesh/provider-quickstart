# Copyright 2025 The Platform Mesh Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL := /usr/bin/env bash

# Tool installation
GO_INSTALL = ./hack/go-install.sh

TOOLS_DIR = hack/tools
GOBIN_DIR := $(abspath $(TOOLS_DIR))

# controller-gen
CONTROLLER_GEN_VER := v0.16.5
CONTROLLER_GEN_BIN := controller-gen
CONTROLLER_GEN := $(GOBIN_DIR)/$(CONTROLLER_GEN_BIN)-$(CONTROLLER_GEN_VER)
export CONTROLLER_GEN

# apigen - generates APIResourceSchemas from CRDs
APIGEN_VER := v0.28.1-0.20251209130449-436a0347809b
APIGEN_BIN := apigen
APIGEN := $(GOBIN_DIR)/$(APIGEN_BIN)-$(APIGEN_VER)
export APIGEN

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GORUN = $(GOCMD) run
GOMOD = $(GOCMD) mod
GOFMT = $(GOCMD) fmt

# Binary names
BINARY_NAME = wild-west
INIT_BINARY_NAME = wild-west-init

# Build directory
BUILD_DIR = bin

.PHONY: all
all: build

## build: Build all binaries
.PHONY: build
build: build-operator build-init

## build-operator: Build the wild-west operator binary
.PHONY: build-operator
build-operator: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wild-west/...

## build-init: Build the init/bootstrap binary
.PHONY: build-init
build-init: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(INIT_BINARY_NAME) ./cmd/init/...

## run: Run the wild-west operator locally
.PHONY: run
run: fmt vet
	$(GORUN) ./cmd/wild-west/main.go --endpointslice=wildwest.platform-mesh.io

## init: Bootstrap provider resources into workspace (requires KUBECONFIG)
.PHONY: init
init: build-init
	$(BUILD_DIR)/$(INIT_BINARY_NAME)

## generate: Generate code (deepcopy, etc.) and KCP resources
.PHONY: generate
generate: $(CONTROLLER_GEN) manifests apiresourceschemas
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."

## manifests: Generate CRD manifests
.PHONY: manifests
manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd paths="./apis/..." output:crd:artifacts:config=config/crds

## apiresourceschemas: Generate APIResourceSchemas from CRDs
.PHONY: apiresourceschemas
apiresourceschemas: manifests $(APIGEN)
	$(APIGEN) --input-dir=config/crds --output-dir=config/kcp

## fmt: Run go fmt
.PHONY: fmt
fmt:
	$(GOFMT) ./...

## vet: Run go vet
.PHONY: vet
vet:
	$(GOCMD) vet ./...

## tidy: Run go mod tidy
.PHONY: tidy
tidy:
	$(GOMOD) tidy

## tools: Install all required tools
.PHONY: tools
tools: $(CONTROLLER_GEN) $(APIGEN)

## help: Display this help
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# Tool installation targets
$(CONTROLLER_GEN):
	GOBIN=$(GOBIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-tools/cmd/$(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_VER)

$(APIGEN):
	GOBIN=$(GOBIN_DIR) $(GO_INSTALL) github.com/kcp-dev/sdk/cmd/$(APIGEN_BIN) $(APIGEN_BIN) $(APIGEN_VER)
