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
ARMAMENT_SYNC_BINARY_NAME = armament-sync

# Build directory
BUILD_DIR = bin

# Image parameters
IMAGE_REGISTRY ?= ghcr.io/platform-mesh
IMAGE_NAME ?= provider-quickstart
IMAGE_TAG ?= dev
IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Portal image parameters
PORTAL_IMAGE_NAME ?= provider-quickstart-portal
PORTAL_IMAGE ?= $(IMAGE_REGISTRY)/$(PORTAL_IMAGE_NAME):$(IMAGE_TAG)
PORTAL_PORT ?= 4200

# Armament-sync image parameters
ARMAMENT_SYNC_IMAGE_NAME ?= provider-quickstart-armament-sync
ARMAMENT_SYNC_IMAGE ?= $(IMAGE_REGISTRY)/$(ARMAMENT_SYNC_IMAGE_NAME):$(IMAGE_TAG)

.PHONY: all
all: build

## build: Build all binaries
.PHONY: build
build: build-operator build-init build-armament-sync

## build-operator: Build the wild-west operator binary
.PHONY: build-operator
build-operator: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wild-west/...

## build-init: Build the init/bootstrap binary
.PHONY: build-init
build-init: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(INIT_BINARY_NAME) ./cmd/init/...

## build-armament-sync: Build the armament-sync controller binary
.PHONY: build-armament-sync
build-armament-sync: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(ARMAMENT_SYNC_BINARY_NAME) ./cmd/armament-sync/...

## run: Run the wild-west operator locally
.PHONY: run
run: fmt vet
	$(GORUN) ./cmd/wild-west/main.go --endpointslice=wildwest.platform-mesh.io

## run-armament-sync: Run the armament-sync controller locally
.PHONY: run-armament-sync
run-armament-sync: fmt vet
	$(GORUN) ./cmd/armament-sync/main.go --sync-interval=30s

## init: Bootstrap provider resources into the workspace pointed to by KUBECONFIG (optional HOST_OVERRIDE)
HOST_OVERRIDE ?=
.PHONY: init
init: build-init
	$(BUILD_DIR)/$(INIT_BINARY_NAME) $(if $(HOST_OVERRIDE),--host-override=$(HOST_OVERRIDE))

## init-seed-workspaces: Create the provider workspace hierarchy from the admin kubeconfig, then bootstrap (requires admin KUBECONFIG, optional HOST_OVERRIDE)
.PHONY: init-seed-workspaces
init-seed-workspaces: build-init
	$(BUILD_DIR)/$(INIT_BINARY_NAME) --seed-workspaces $(if $(HOST_OVERRIDE),--host-override=$(HOST_OVERRIDE))

## generate: Generate code (deepcopy, etc.) and kcp resources
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
	$(APIGEN) --input-dir=config/crds --output-dir=config/kcp --preserve-resources

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

## image-build: Build controller container image locally
.PHONY: image-build
image-build:
	docker build -t $(IMAGE) -f deploy/Dockerfile .

## image-push: Push controller container image to registry
.PHONY: image-push
image-push: image-build
	docker push $(IMAGE)

## portal-image-build: Build portal container image locally
.PHONY: portal-image-build
portal-image-build:
	docker build -t $(PORTAL_IMAGE) -f deploy/portal.Dockerfile .

## portal-image-push: Push portal container image to registry
.PHONY: portal-image-push
portal-image-push: portal-image-build
	docker push $(PORTAL_IMAGE)

## armament-sync-image-build: Build armament-sync container image locally
.PHONY: armament-sync-image-build
armament-sync-image-build:
	docker build -t $(ARMAMENT_SYNC_IMAGE) -f deploy/armament-sync.Dockerfile .

## armament-sync-image-push: Push armament-sync container image to registry
.PHONY: armament-sync-image-push
armament-sync-image-push: armament-sync-image-build
	docker push $(ARMAMENT_SYNC_IMAGE)

## images: Build all container images
.PHONY: images
images: image-build portal-image-build armament-sync-image-build

## images-push: Push all container images
.PHONY: images-push
images-push: image-push portal-image-push armament-sync-image-push

# Kind cluster parameters
KIND_CLUSTER ?= platform-mesh

## kind-load: Load controller image into kind cluster
.PHONY: kind-load
kind-load: image-build
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER)

## kind-load-portal: Load portal image into kind cluster
.PHONY: kind-load-portal
kind-load-portal: portal-image-build
	kind load docker-image $(PORTAL_IMAGE) --name $(KIND_CLUSTER)

## kind-load-armament-sync: Load armament-sync image into kind cluster
.PHONY: kind-load-armament-sync
kind-load-armament-sync: armament-sync-image-build
	kind load docker-image $(ARMAMENT_SYNC_IMAGE) --name $(KIND_CLUSTER)

## kind-load-all: Load all images into kind cluster
.PHONY: kind-load-all
kind-load-all: kind-load kind-load-portal kind-load-armament-sync

## portal-run: Run portal container locally (accessible at http://localhost:$(PORTAL_PORT))
.PHONY: portal-run
portal-run:
	docker run --rm -p $(PORTAL_PORT):8080 $(PORTAL_IMAGE)

## portal-run-detached: Run portal container in background
.PHONY: portal-run-detached
portal-run-detached:
	docker run -d --rm --name wildwest-portal -p $(PORTAL_PORT):80 $(PORTAL_IMAGE)
	@echo "Portal running at http://localhost:$(PORTAL_PORT)"
	@echo "Stop with: docker stop wildwest-portal"

## portal-stop: Stop the portal container
.PHONY: portal-stop
portal-stop:
	docker stop wildwest-portal

## tools: Install all required tools
.PHONY: tools
tools: $(CONTROLLER_GEN) $(APIGEN)

# OCM parameters
OCM ?= ocm
OCM_REPO ?= ghcr.io/platform-mesh
# Component name as declared in constructor/component-constructor.yaml.
OCM_COMPONENT ?= github.com/platform-mesh/provider-quickstart
OCM_CTF ?= .ocm/transport.ctf
VERSION ?= 0.0.0-dev
CHART_VERSION ?= $(VERSION)
IMAGE_VERSION ?= $(VERSION)

# Helm chart publishing parameters
HELM ?= helm
# Charts are published under this repo's own GHCR namespace (self-contained,
# alongside the container images) rather than the shared helm-charts registry.
HELM_REPO ?= ghcr.io/platform-mesh/provider-quickstart/charts
# Deployable charts published as standalone OCI Helm artifacts. These are consumed
# by the platform-mesh-operator ManagedProvider machinery (Flux OCIRepository ->
# HelmRelease); see config/pm/README.md.
HELM_CHARTS ?= wildwest-controller wildwest-portal wildwest-armament-sync
# OCI registry tag for the referenced images (free-form, e.g. "latest" or "0.1.0").
# Defaults to "latest" so local builds resolve against an existing tag; CI sets it to the release tag.
OCI_TAG ?= latest

## ocm-build: Build OCM component archive (CTF) from constructor/component-constructor.yaml
.PHONY: ocm-build
ocm-build:
	mkdir -p $(dir $(OCM_CTF))
	rm -rf $(OCM_CTF)
	$(OCM) add components -c --templater=go --file $(OCM_CTF) constructor/component-constructor.yaml -- \
	  VERSION=$(VERSION) \
	  CHART_VERSION=$(CHART_VERSION) \
	  IMAGE_VERSION=$(IMAGE_VERSION) \
	  OCI_TAG=$(OCI_TAG)

## ocm-push: Transfer the OCM component archive to $(OCM_REPO)
.PHONY: ocm-push
ocm-push: ocm-build
	$(OCM) transfer ctf --overwrite $(OCM_CTF) $(OCM_REPO)

## ocm-describe: Print the locally built component descriptor
.PHONY: ocm-describe
ocm-describe: ocm-build
	$(OCM) get componentversions --repo $(OCM_CTF) -o yaml

## ocm-get: Inspect the PUBLISHED component in $(OCM_REPO) (pass VERSION=<tag>)
.PHONY: ocm-get
ocm-get:
	$(OCM) get cv $(OCM_REPO)//$(OCM_COMPONENT):$(VERSION) -o yaml

## helm-push: Package and push deployable Helm charts to $(HELM_REPO) as OCI artifacts
.PHONY: helm-push
helm-push:
	mkdir -p $(BUILD_DIR)/charts
	@for chart in $(HELM_CHARTS); do \
	  echo "==> packaging $$chart $(CHART_VERSION)"; \
	  $(HELM) dependency build deploy/helm/$$chart || exit 1; \
	  $(HELM) package deploy/helm/$$chart \
	    --version $(CHART_VERSION) \
	    --app-version $(IMAGE_VERSION) \
	    --destination $(BUILD_DIR)/charts || exit 1; \
	  echo "==> pushing $$chart-$(CHART_VERSION).tgz to oci://$(HELM_REPO)"; \
	  $(HELM) push $(BUILD_DIR)/charts/$$chart-$(CHART_VERSION).tgz oci://$(HELM_REPO) || exit 1; \
	done

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
