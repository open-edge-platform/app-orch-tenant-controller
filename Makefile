# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SHELL	:= bash -eu -o pipefail

PKG               := github.com/open-edge-platform/app-orch-tenant-controller

# GO variables
GOARCH	:= $(shell go env GOARCH)
GOCMD   := GOPRIVATE="github.com/open-edge-platform/*" go
GOLANG_COVER_VERSION = v0.2.0
GOLANG_GOCOVER_COBERTURA_VERSION = v1.2.0
GOPATH := $(shell go env GOPATH)

# Code Versions
VERSION              := $(shell cat VERSION)
GIT_HASH_SHORT       := $(shell git rev-parse --short=8 HEAD)
VERSION_DEV_SUFFIX   := ${GIT_HASH_SHORT}

PUBLISH_NAME            ?= app-orch-tenant-controller
PUBLISH_REPOSITORY      ?= edge-orch
PUBLISH_REGISTRY        ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
PUBLISH_SUB_PROJ        ?= app
PUBLISH_CHART_PREFIX    ?= charts

DOCKER_TAG              := $(PUBLISH_REGISTRY)/$(PUBLISH_REPOSITORY)/$(PUBLISH_SUB_PROJ)/$(PUBLISH_NAME):$(VERSION)
DOCKER_BUILD_COMMAND    := docker buildx build

ifdef PLATFORM
	DOCKER_BUILD_ARGS := --platform $(PLATFORM)
endif

# Add an identifying suffix for `-dev` builds only.
# Release build versions are verified as unique by the CI build process.
ifeq ($(findstring -dev,$(VERSION)), -dev)
	VERSION := $(VERSION)-$(VERSION_DEV_SUFFIX)
endif

CONFIG_PROVISIONER_VERSION    ?= ${VERSION}
DOCKER_BUILD_ARGS += -t $(PUBLISH_NAME):$(VERSION)

CHART_NAMESPACE				?= orch-app
CHART_APP_VERSION			?= "${CONFIG_PROVISIONER_VERSION}"
CHART_VERSION				?= $(shell yq -r .version ${CHART_PATH}/Chart.yaml)
CHART_BUILD_DIR				?= ./build/_output/
CHART_PATH					?= ./deploy/charts/${PUBLISH_NAME}

CONTROLLER_TOOLS_VERSION ?= v0.10.0
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CRD_OPTIONS ?= "crd:trivialVersions=true"
CODE_GENERATOR_TAG ?= v0.30.0

MGMT_NAME        ?= kind
MGMT_CLUSTER    ?= kind-${MGMT_NAME}
CODER_DIR ?= ~/orch-deploy
CONFIG_PROVISIONER_HELM_PKG ?= $(MAKEDIR)/${CHART_BUILD_DIR}${PUBLISH_NAME}-${CHART_VERSION}.tgz

GOPATH := $(shell go env GOPATH)
GOLANG_COVER_VERSION             = v0.2.0
GOLANG_GOCOVER_COBERTURA_VERSION = v1.2.0

include common.mk

## Virtual environment name
VENV_NAME = venv-env

.PHONY: all
all: build go-lint test
	@# Help: Runs build, lint, test stages

# Yamllint variables
YAML_FILES         := $(shell find . -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
YAML_IGNORE        := .cache, vendor, ci, .github/workflows, $(VENV_NAME), internal/plugins/testdata/extensions/*.yaml

MAKEDIR          := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))

.PHONY: go-tidy
go-tidy: ## Run go mod tidy
	$(GOCMD) mod tidy

.PHONY: go-lint-fix
go-lint-fix: ## Apply automated lint/formatting fixes to go files
	golangci-lint run --fix --config .golangci.yml

.PHONY: go-lint
go-lint: $(OUT_DIR) ## Run go lint
	golangci-lint --version
	golangci-lint run --timeout 10m $(LINT_DIRS) --config .golangci.yml

.PHONY: build
build: go-tidy go-build

.PHONY: go-build
go-build: ## Runs build stage
	@echo "---MAKEFILE BUILD---"
	$(GOCMD) build -o build/_output/provisioner ./cmd/provisioner
	@echo "---END MAKEFILE Build---"

.PHONY: go-test
go-test: ## Runs test stage
	@echo "---MAKEFILE TEST---"
	$(GOCMD) test -race -gcflags=-l `go list  $(PKG)/pkg/... | grep -v "/mocks" | grep -v "/test/"`
	@echo "---END MAKEFILE TEST---"

.PHONY: go-cover-dependency
go-cover-dependency: ## installs the gocover tool
	go tool cover -V || go install golang.org/x/tools/cmd/cover@${GOLANG_COVER_VERSION}
	go install github.com/boumenot/gocover-cobertura@${GOLANG_GOCOVER_COBERTURA_VERSION}

.PHONY: hadolint
hadolint: ## Runs hadolint
	hadolint --ignore DL3059 build/Dockerfile

.PHONY: lint
lint: yamllint go-lint hadolint mdlint ## Runs lint stage
	golangci-lint run --timeout 10m
	hadolint --ignore DL3059 build/Dockerfile
	helm lint deploy/charts/app-orch-tenant-controller

.PHONY: test
test: ## Runs test stage
	$(GOCMD) test -race -gcflags=-l `go list $(PKG)/internal/...`

.PHONY: coverage
coverage: go-cover-dependency ## Runs coverage stage
	$(GOCMD) test -gcflags=-l `go list $(PKG)/cmd/... $(PKG)/internal/... | grep -v "/mocks" | grep -v "/test/"` -v -coverprofile=coverage.txt -covermode count
	${GOPATH}/bin/gocover-cobertura < coverage.txt > coverage.xml

.PHONY: list
list: ## displays make targets
	help

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: vendor
vendor: ## go mod vendor
	$(GOCMD) mod vendor

.PHONY: docker-load
docker-load: ## Build and load the Docker image
docker-load: DOCKER_BUILD_ARGS += --load
docker-load: docker-image

.PHONY: docker-image
docker-image: ## Vendor, build the Docker image and clean
docker-image: vendor docker-build

.PHONY: docker-build
docker-build: ## Build the Docker image
docker-build: vendor
	$(DOCKER_BUILD_COMMAND) . $(DOCKER_BUILD_ARGS) -f build/Dockerfile

.PHONY: docker-push
docker-push: docker-build ##Push the docker image to the target registry
	aws ecr create-repository --region us-west-2 --repository-name $(PUBLISH_REPOSITORY)/$(PUBLISH_SUB_PROJ)/$(PUBLISH_NAME) || true
	docker tag $(PUBLISH_NAME):$(VERSION) $(DOCKER_TAG)
	docker push $(DOCKER_TAG)

docker-list: ## Print name of docker container image
	@echo "images:"
	@echo "  $(PUBLISH_NAME):"
	@echo "    name: '$(DOCKER_TAG)'"
	@echo "    version: '$(VERSION)'"
	@echo "    gitTagPrefix: 'v'"
	@echo "    buildTarget: 'docker-build'"

.PHONY: chart-clean
chart-clean: ## Cleans the build directory of the helm chart
	rm -rf ${CHART_BUILD_DIR}/*.tgz
	yq eval -i 'del(.annotations.revision)' ${CHART_PATH}/Chart.yaml
	yq eval -i 'del(.annotations.created)' ${CHART_PATH}/Chart.yaml

.PHONY: chart-meta
chart-meta: ## Sets Chart meta-data according to build settings
	yq eval -i '.annotations.revision = "${LABEL_REVISION}"' ${CHART_PATH}/Chart.yaml
	yq eval -i '.annotations.created = "${LABEL_CREATED}"' ${CHART_PATH}/Chart.yaml
	yq eval -i '.appVersion = ${CHART_APP_VERSION}' ${CHART_PATH}/Chart.yaml
	yq eval -i '.version = "${CONFIG_PROVISIONER_VERSION}"' ${CHART_PATH}/Chart.yaml

.PHONY: chart
chart: chart-clean chart-meta ## Builds the tenant controller helm chart
	yq eval -i '.annotations.revision = "${DOCKER_LABEL_VCS_REF}"' ${CHART_PATH}/Chart.yaml
	yq eval -i '.annotations.created = "${DOCKER_LABEL_BUILD_DATE}"' ${CHART_PATH}/Chart.yaml
	helm package \
		--app-version=${CHART_APP_VERSION} \
		--debug \
		--dependency-update \
		--destination ${CHART_BUILD_DIR} \
		${CHART_PATH}

.PHONY: chart-install-kind
chart-install-kind: chart docker-build ## Installs the tenant controller helm chart in the kind cluster
	helm upgrade --install -n ${CHART_NAMESPACE} tenant-controller \
			--wait --timeout 300s \
			--set logging.rootLogger.level=DEBUG \
			deploy/charts/app-orch-tenant-controller

.PHONY: helm-build
helm-build: chart ## Builds the helm chart artifact

.PHONY: helm-push
helm-push: helm-build ## Push helm charts.
	@# Help: Pushes the helm chart
	aws ecr create-repository --region us-west-2 --repository-name $(PUBLISH_REPOSITORY)/$(PUBLISH_SUB_PROJ)/$(PUBLISH_CHART_PREFIX)/$(PUBLISH_NAME) || true
	helm push ${CHART_BUILD_DIR}${PUBLISH_NAME}-[0-9]*.tgz oci://$(PUBLISH_REGISTRY)/$(PUBLISH_REPOSITORY)/$(PUBLISH_SUB_PROJ)/$(PUBLISH_CHART_PREFIX)

helm-list: ## List helm charts, tag format, and versions in YAML format
	@echo "charts:" ;\
  echo "  $(PUBLISH_NAME):" ;\
  echo -n "    "; grep "^version" "${CHART_PATH}/Chart.yaml"  ;\
  echo "    gitTagPrefix: 'v'" ;\
  echo "    outDir: '${CHART_BUILD_DIR}'" ;\

.PHONY: kind
kind: ## Build the Docker image and load it into kind
kind: docker-image docker-load
kind:
	kind load docker-image $(DOCKER_IMAGE_NAME):$(VERSION)


.PHONY: coder-rebuild
coder-rebuild: ## Rebuild the TC from source and redeploy
	make docker-image
	kind load docker-image -n ${MGMT_NAME} $(DOCKER_IMAGE_NAME):$(VERSION)
	kubectl config use-context ${MGMT_CLUSTER}
	kubectl -n ${CHART_NAMESPACE} delete pod -l app=config-provisioner

.PHONY: coder-redeploy
coder-redeploy: kind chart ## Installs the helm chart in the kind cluster
	kubectl config use-context ${MGMT_CLUSTER}
	kubectl patch application -n dev root-app --type=merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
	kubectl delete application -n dev config-provisioner --ignore-not-found=true
	helm upgrade --install -n vcm-system config-provisioner -f $(CODER_DIR)/argocd/applications/configs/config-provisioner.yaml  $(CONFIG_PROVISIONER_HELM_PKG)
	helm -n vcm-system ls


.PHONY: clean
clean: clean-all
	$(GOCMD) clean -testcache
	rm -rf vendor build/_output

.PHONY: dependency-check
dependency-check: ## Unsupported target
	echo '"make $@" is unsupported'
