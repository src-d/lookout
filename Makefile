# Package configuration
PROJECT = lookout
COMMANDS = cmd/lookout
DEPENDENCIES = \
	gopkg.in/src-d/go-kallax.v1 \
	github.com/jteeuwen/go-bindata

# Backend services
POSTGRESQL_VERSION = 9.6
MIGRATIONS_PATH = store/migrations

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_BRANCH ?= v1
CI_PATH ?= .ci
MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --depth 1 -b $(CI_BRANCH) $(CI_REPOSITORY) $(CI_PATH);
-include $(MAKEFILE)

# Main dockerfile
DOCKERFILES=./docker/Dockerfile:$(PROJECT)

# Environment
OS := $(shell uname)
CONFIG_FILE := config.yml

# SDK binaries
DUMMY_BIN := $(BIN_PATH)/dummy
LOOKOUT_BIN := $(BIN_PATH)/lookout

# Tools
BINDATA := go-bindata

.PHONY: bindata
bindata:
	chmod -R go=r $(MIGRATIONS_PATH); \
	$(BINDATA) \
		-o store/bindata.go \
		-pkg store \
		-prefix '$(MIGRATIONS_PATH)/' \
		-modtime 1 \
		$(MIGRATIONS_PATH)/...

# Protoc
PROTOC_DIR ?= ./protoc
PROTOC := $(PROTOC_DIR)/bin/protoc
PROTOC_VERSION := 3.6.0
ifeq ($(OS),Darwin)
PROTOC_OS := "osx"
else
PROTOC_OS := "linux"
endif
PROTOC_ZIP_NAME := protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-x86_64.zip
PROTOC_URL := https://github.com/google/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP_NAME)

# Generate go code from proto files
.PHONY: protogen
protogen: install-protoc
	$(GOCMD) install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofaster
	$(PROTOC) \
		-I sdk \
		--gogofaster_out=plugins=grpc,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:pb \
sdk/*.proto

.PHONY: install-protoc
install-protoc: $(PROTOC)
$(PROTOC):
	mkdir -p $(PROTOC_DIR)
	wget $(PROTOC_URL) -O $(PROTOC_DIR)/$(PROTOC_ZIP_NAME)
	unzip  -d $(PROTOC_DIR) $(PROTOC_DIR)/$(PROTOC_ZIP_NAME)
	rm $(PROTOC_DIR)/$(PROTOC_ZIP_NAME)

GOTEST_INTEGRATION = $(GOTEST) -tags=integration

# Integration test for sdk client
.PHONY: test-sdk
test-sdk: clean-sdk build-sdk
	DUMMY_BIN=$(PWD)/$(DUMMY_BIN) \
	LOOKOUT_BIN=$(PWD)/$(LOOKOUT_BIN) \
	$(GOTEST_INTEGRATION) github.com/src-d/lookout/cmd/sdk-test

# Same as test-sdk, but skipping tests that require a bblfshd server
.PHONY: test-sdk-short
test-sdk-short: clean-sdk build-sdk
	DUMMY_BIN=$(PWD)/$(DUMMY_BIN) \
	LOOKOUT_BIN=$(PWD)/$(LOOKOUT_BIN) \
	$(GOTEST_INTEGRATION) -test.short github.com/src-d/lookout/cmd/sdk-test

# Integration test for lookout serve
.PHONY: test-json
test-json: clean-sdk build-sdk
	DUMMY_BIN=$(PWD)/$(DUMMY_BIN) \
	LOOKOUT_BIN=$(PWD)/$(LOOKOUT_BIN) \
	$(GOTEST_INTEGRATION) github.com/src-d/lookout/cmd/server-test

# Build sdk client and dummy analyzer
.PHONY: build-sdk
build-sdk: $(DUMMY_BIN) $(LOOKOUT_BIN)
$(LOOKOUT_BIN):
	$(GOBUILD) -o "$(LOOKOUT_BIN)" ./cmd/lookout
$(DUMMY_BIN):
	$(GOBUILD) -o "$(DUMMY_BIN)" ./cmd/dummy

.PHONY: clean-sdk
clean-sdk:
	rm -f $(DUMMY_BIN)
	rm -f $(LOOKOUT_BIN)

.PHONY: dry-run
dry-run: $(CONFIG_FILE)
	go run cmd/lookout/*.go serve --dry-run github.com/src-d/lookout
$(CONFIG_FILE):
	cp "$(CONFIG_FILE).tpl" $(CONFIG_FILE)

# Builds build/lookout_sdk_*.tar.gz with the lookout bin and sdk dir
.PHONY: packages-sdk
packages-sdk: PROJECT = lookout_sdk
packages-sdk: build
	@for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			cp -r sdk $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/; \
		done; \
	done; \
	cd $(BUILD_PATH); \
	for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			TAR_VERSION=`echo $(VERSION) | tr "/" "-"`; \
			tar -cvzf $(PROJECT)_$${TAR_VERSION}_$${os}_$${arch}.tar.gz $(PROJECT)_$${os}_$${arch}/; \
		done; \
	done

# TODO: remove when https://github.com/src-d/ci/pull/84 is merged
.PHONY: godep
GODEP ?= $(CI_PATH)/dep
godep:
	export INSTALL_DIRECTORY=$(CI_PATH) ; \
	test -f $(GODEP) || \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | bash ; \
	$(GODEP) ensure -v
