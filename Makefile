
# Image URL to use all building/pushing image targets
IMG ?= knode:latest

# Directories
TOOLS_DIR := $(PWD)/hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
BIN_DIR := bin

# Binaries
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
KUSTOMIZE := $(TOOLS_BIN_DIR)/kustomize

all: knode

# Run tests
test: lint
	go test ./... -coverprofile cover.out

# Binaries

knode: lint-full
	go build -o bin/knode main.go

$(GOLANGCI_LINT): $(TOOLS_DIR)/go.mod # Build golangci-lint from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(KUSTOMIZE): $(TOOLS_DIR)/go.mod # Build kustomize from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/kustomize sigs.k8s.io/kustomize/kustomize/v3

# Deploy in the configured Kubernetes cluster in ~/.kube/config
deploy: $(KUSTOMIZE)
	cd config/knode && $(KUSTOMIZE) edit set image knode=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Linting

.PHONY: lint lint-full
lint: $(GOLANGCI_LINT) ## Lint codebase
	$(GOLANGCI_LINT) run -v

lint-full: $(GOLANGCI_LINT) ## Run slower linters to detect possible issues
	$(GOLANGCI_LINT) run -v --fast=false

# Build the docker image
docker-build:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# Release

RELEASE_DIR := out

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)/

.PHONY: release
release: release-containers release-manifests

.PHONY: release-manifests
release-manifests: $(KUSTOMIZE) $(RELEASE_DIR)
	cd config/knode && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > $(RELEASE_DIR)/knode-components.yaml

.PHONY: release-containers
release-containers: $(RELEASE_DIR)
	$(MAKE) docker-build docker-push

