# Package configuration
PROJECT = lookout
COMMANDS = cmd/lookout

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_BRANCH ?= v1
CI_PATH ?= .ci
MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --depth 1 -b $(CI_BRANCH) $(CI_REPOSITORY) $(CI_PATH);
-include $(MAKEFILE)

# Environment
OS := $(shell uname)
CONFIG_FILE := config.yml

# SDK binaries
DUMMY_BIN := $(BIN_PATH)/dummy
LOOKOUT_BIN := $(BIN_PATH)/lookout

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

# Integration test for sdk client
.PHONY: test-sdk
test-sdk: clean-sdk build-sdk
	$(DUMMY_BIN) serve &>/dev/null & \
	PID=$$!; \
	$(LOOKOUT_BIN) review ipv4://localhost:10302 2>&1 | grep "posting analysis"; \
	if [ $$? != 0 ] ; then \
		echo "review test failed"; \
		kill $$PID; \
		exit 1; \
	fi; \
	$(LOOKOUT_BIN) push ipv4://localhost:10302 2>&1 | grep "dummy comment for push event"; \
	if [ $$? != 0 ] ; then \
		echo "push test failed"; \
		kill $$PID; \
		exit 1; \
	fi; \
	kill $$PID || true ; \

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
