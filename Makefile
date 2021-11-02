PREFIX := /usr
BINDIR := $(PREFIX)/bin
CONFIG_DIR := /etc/isula-build
LOCAL_CONF_PREFIX := cmd/daemon/config
CONFIG_FILE := configuration.toml
POLICY_FILE := policy.json
REGIST_FILE := registries.toml
STORAGE_FILE := storage.toml

SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
GIT_COMMIT ?= $(if $(shell git rev-parse --short HEAD),$(shell git rev-parse --short HEAD),$(shell cat ./git-commit | head -c 7))
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
VERSION := $(shell cat ./VERSION)
ARCH := $(shell arch)

EXTRALDFLAGS :=
LDFLAGS := -X isula.org/isula-build/pkg/version.GitCommit=$(GIT_COMMIT) \
           -X isula.org/isula-build/pkg/version.BuildInfo=$(SOURCE_DATE_EPOCH) \
           -X isula.org/isula-build/pkg/version.Version=$(VERSION) \
           $(EXTRALDFLAGS)
BUILDTAGS := seccomp
BUILDFLAGS := -tags "$(BUILDTAGS)"
TMPDIR := /tmp/isula_build_tmpdir
BEFLAG := -tmpdir=${TMPDIR}
SAFEBUILDFLAGS := -buildid=IdByIsula -buildmode=pie -extldflags=-ftrapv -extldflags=-zrelro -extldflags=-znow $(BEFLAG) $(LDFLAGS)
STATIC_LDFLAGS := -linkmode=external -extldflags=-static

IMAGE_BUILDARGS := $(if $(http_proxy), --build-arg http_proxy=$(http_proxy))
IMAGE_BUILDARGS += $(if $(https_proxy), --build-arg https_proxy=$(https_proxy))
IMAGE_BUILDARGS += --build-arg arch=$(ARCH)

IMAGE_NAME := isula-build-dev

GO := go
# test for go module support
ifeq ($(shell go help mod >/dev/null 2>&1 && echo true), true)
export GO_BUILD=GO111MODULE=on; $(GO) build -mod=vendor
else
export GO_BUILD=$(GO) build
endif

##@ Help
.PHONY: help
help: ## Display the help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: all ## Build both isula-build and isula-builder
all: isula-build isula-builder

.PHONY: isula-build
isula-build: ./cmd/cli ## Build isula-build only
	@echo "Making isula-build..."
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o bin/isula-build $(BUILDFLAGS) ./cmd/cli
	@echo "isula-build done!"

.PHONY: isula-builder
isula-builder: ./cmd/daemon ## Build isula-builder only
	@echo "Making isula-builder..."
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o bin/isula-builder $(BUILDFLAGS) ./cmd/daemon
	@echo "isula-builder done!"

.PHONY: safe
safe: ## Build binary with secure compile flag enabled
	@echo "Safe building isula-build..."
	mkdir -p ${TMPDIR}
	$(GO_BUILD) -ldflags '$(SAFEBUILDFLAGS) $(STATIC_LDFLAGS)' -o bin/isula-build $(BUILDFLAGS) ./cmd/cli 2>/dev/null
	$(GO_BUILD) -ldflags '$(SAFEBUILDFLAGS)' -o bin/isula-builder $(BUILDFLAGS) ./cmd/daemon
	@echo "Safe build isula-build done!"

.PHONY: debug
debug: ## Build binary with debug info inside
	@echo "Debug building isula-build..."
	@cp -f ./hack/profiling ./daemon/profiling.go
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -gcflags="all=-N -l" -o bin/isula-build $(BUILDFLAGS) ./cmd/cli
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -gcflags="all=-N -l" -o bin/isula-builder $(BUILDFLAGS) ./cmd/daemon
	@rm -f ./daemon/profiling.go
	@echo "Debug build isula-build done!"

.PHONY: install
install: ## Install binary and configs
	install -D -m0550 bin/isula-build $(BINDIR)
	install -D -m0550 bin/isula-builder $(BINDIR)
	@( getent group isula > /dev/null ) || ( groupadd --system isula )
	@[ ! -d ${CONFIG_DIR}/${CONFIG_FILE} ] && install -dm0650 ${CONFIG_DIR}
	@( [ -f ${CONFIG_DIR}/${CONFIG_FILE} ] && printf "%-20s %s\n" "${CONFIG_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${CONFIG_FILE} ${CONFIG_DIR}/${CONFIG_FILE}
	@( [ -f ${CONFIG_DIR}/${POLICY_FILE} ] && printf "%-20s %s\n" "${POLICY_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${POLICY_FILE} ${CONFIG_DIR}/${POLICY_FILE}
	@( [ -f ${CONFIG_DIR}/${REGIST_FILE} ] && printf "%-20s %s\n" "${REGIST_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${REGIST_FILE} ${CONFIG_DIR}/${REGIST_FILE}
	@( [ -f ${CONFIG_DIR}/${STORAGE_FILE} ] && printf "%-20s %s\n" "${STORAGE_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${STORAGE_FILE} ${CONFIG_DIR}/${STORAGE_FILE}


##@ Test

tests: test-base test-unit test-integration ## Test all

.PHONY: test-base
test-base: ## Test base case
	@echo "Base test starting..."
	@./tests/test.sh base
	@echo "Base test done!"

.PHONY: test-unit
test-unit: ## Test unit case
	@echo "Unit test starting..."
	@./hack/unit_test.sh
	@echo "Unit test done!"

.PHONY: test-integration
test-integration: ## Test integration case
	@echo "Integration test starting..."
	@./tests/test.sh integration
	@echo "Integration test done!"

.PHONY: test-unit-cover
test-unit-cover: ## Test unit case and generate coverage
	@echo "Unit test cover starting..."
	@./hack/unit_test.sh cover
	@echo "Unit test cover done!"

.PHONY: test-integration-cover
test-integration-cover: ## Test integration case and generate coverage
	@echo "Integration test cover starting..."
	@./hack/integration_coverage.sh
	@echo "Integration test cover done!"

.PHONY: test-cover
test-cover: test-integration-cover test-unit-cover ## Test both unit and integration case and generate unity coverage
	@echo "Test cover starting..."
	@./hack/all_coverage.sh
	@echo "Test cover done!"

##@ Development

.PHONY: build-image
build-image: ## Build protobuf compile environment container image
	isula-build ctr-img build -f Dockerfile.proto ${IMAGE_BUILDARGS} -o isulad:${IMAGE_NAME}:latest .

.PHONY: proto
proto: ## Generate protobuf file
	@echo "Generating protobuf..."
	isula run -i --rm --runtime runc -v ${PWD}:/go/src/isula.org/isula-build ${IMAGE_NAME} ./hack/generate_proto.sh
	@echo "Protobuf files have been generated!"

.PHONY: check
check: ## Static check for current commit
	@echo "Static check start for last commit"
	@./hack/static_check.sh last
	@echo "Static check last commit finished"

.PHONY: checkall
checkall: ## Static check for whole project
	@echo "Static check start for whole project"
	@./hack/static_check.sh all
	@echo "Static check project finished"

.PHONY: clean
clean: ## Clean project
	rm -rf ./bin
