SHELL := /bin/bash

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# options for generating crds with controller-gen
CONTROLLER_GEN="${GOBIN}/controller-gen"
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

.EXPORT_ALL_VARIABLES:

default: build

fmt: ## Run go fmt against code.
	go fmt ./cmd/...

vet: ## Run go vet against code.
	go vet ./cmd/...

test: fmt vet ## Run tests.
	go test -v ./cmd/... -coverprofile cover.out

build:
	env GOOS=linux GOARCH=arm64 go build -mod=vendor -o out/multi-arch-resolver ./cmd/resolver

clean:
	rm -rf out

dev-image:
	docker build . -t quay.io/$(QUAY_USERNAME)/multi-arch-resolver:dev
	docker push quay.io/$(QUAY_USERNAME)/multi-arch-resolver:dev

dev: dev-image
	./deploy/development.sh

dev-minikube:
	eval $(minikube -p minikube docker-env)
	docker build . -t quay.io/$(QUAY_USERNAME)/multi-arch-resolver:dev
	./deploy/minikube-development.sh

