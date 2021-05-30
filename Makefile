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
SAFEBUILDFLAGS := -buildid=IdByIsula -buildmode=pie -extldflags=-ftrapv -extldflags=-static -extldflags=-zrelro -extldflags=-znow $(LDFLAGS) $(BEFLAG)

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

all: isula-build isula-builder

.PHONY: isula-build
isula-build: ./cmd/cli
	@echo "Making isula-build..."
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o bin/isula-build $(BUILDFLAGS) ./cmd/cli
	@echo "isula-build done!"

.PHONY: isula-builder
isula-builder: ./cmd/daemon
	@echo "Making isula-builder..."
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o bin/isula-builder $(BUILDFLAGS) ./cmd/daemon
	@echo "isula-builder done!"

.PHONY: safe
safe:
	@echo "Safe building isula-build..."
	mkdir -p ${TMPDIR}
	$(GO_BUILD) -ldflags '$(SAFEBUILDFLAGS)' -o bin/isula-build $(BUILDFLAGS) ./cmd/cli
	$(GO_BUILD) -ldflags '$(SAFEBUILDFLAGS)' -o bin/isula-builder $(BUILDFLAGS) ./cmd/daemon
	@echo "Safe build isula-build done!"

.PHONY: debug
debug:
	@echo "Debug building isula-build..."
	@cp -f ./hack/profiling ./daemon/profiling.go
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -gcflags="all=-N -l" -o bin/isula-build $(BUILDFLAGS) ./cmd/cli
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -gcflags="all=-N -l" -o bin/isula-builder $(BUILDFLAGS) ./cmd/daemon
	@rm -f ./daemon/profiling.go
	@echo "Debug build isula-build done!"

.PHONY: build-image
build-image:
	isula-build ctr-img build -f Dockerfile.proto ${IMAGE_BUILDARGS} -o isulad:${IMAGE_NAME}:latest .

tests: test-base test-unit test-integration

.PHONY: test-base
test-base:
	@echo "Base test starting..."
	@./tests/test.sh base
	@echo "Base test done!"

.PHONY: test-unit
test-unit:
	@echo "Unit test starting..."
	@./hack/unit_test.sh
	@echo "Unit test done!"

.PHONY: test-integration
test-integration:
	@echo "Integration test starting..."
	@./tests/test.sh integration
	@echo "Integration test done!"

.PHONY: proto
proto:
	@echo "Generating protobuf..."
	isula run -i --rm --runtime runc -v ${PWD}:/go/src/isula.org/isula-build ${IMAGE_NAME} ./hack/generate_proto.sh
	@echo "Protobuf files have been generated!"

.PHONY: install
install:
	install -D -m0551 bin/isula-build $(BINDIR)
	install -D -m0550 bin/isula-builder $(BINDIR)
	@( getent group isula > /dev/null ) || ( groupadd --system isula )
	@[ ! -d ${CONFIG_DIR}/${CONFIG_FILE} ] && install -dm0650 ${CONFIG_DIR}
	@( [ -f ${CONFIG_DIR}/${CONFIG_FILE} ] && printf "%-20s %s\n" "${CONFIG_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${CONFIG_FILE} ${CONFIG_DIR}/${CONFIG_FILE}
	@( [ -f ${CONFIG_DIR}/${POLICY_FILE} ] && printf "%-20s %s\n" "${POLICY_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${POLICY_FILE} ${CONFIG_DIR}/${POLICY_FILE}
	@( [ -f ${CONFIG_DIR}/${REGIST_FILE} ] && printf "%-20s %s\n" "${REGIST_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${REGIST_FILE} ${CONFIG_DIR}/${REGIST_FILE}
	@( [ -f ${CONFIG_DIR}/${STORAGE_FILE} ] && printf "%-20s %s\n" "${STORAGE_FILE}" "already exist in ${CONFIG_DIR}, please replace it manually." ) || install -D -m0600 ${LOCAL_CONF_PREFIX}/${STORAGE_FILE} ${CONFIG_DIR}/${STORAGE_FILE}

.PHONY: checkall
checkall:
	@echo "Static check start for whole project"
	@./hack/static_check.sh all
	@echo "Static check project finished"
.PHONY: check
check:
	@echo "Static check start for last commit"
	@./hack/static_check.sh last
	@echo "Static check last commit finished"

.PHONY: clean
clean:
	rm -rf ./bin
