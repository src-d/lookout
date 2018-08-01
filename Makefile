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

# Main dockerfile
DOCKERFILES=./docker/Dockerfile:$(PROJECT)

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
	DUMMY_BIN=$(DUMMY_BIN) LOOKOUT_BIN=$(LOOKOUT_BIN) $(GOCMD) run sdk-test/main.go

# Integration test for lookout serve
.PHONY: test-json
test-json: build-sdk
	$(DUMMY_BIN) serve &>/dev/null & \
	DUMMY_PID=$$!; \
	cat fixtures/events.jsonl | $(LOOKOUT_BIN) serve --provider json dummy-repo-url > serve.txt 2> serve-err.txt & \
	LOOKOUT_PID=$$!; \
	for i in `seq 0 120`; do \
		sleep 1; \
		grep '{"file":"provider/common.go","text":"The file has increased in 5 lines."}' serve.txt; \
		if [ $$? = 0 ] ; then \
			kill $$LOOKOUT_PID; \
			kill $$DUMMY_PID; \
			exit 0; \
		fi; \
	done; \
	echo "timeout reached, inspect serve.txt and serve-err.txt for details"; \
	echo -e "\nserve.txt:"; \
	cat serve.txt; \
	echo -e "\nserve-err.txt:"; \
	cat serve-err.txt; \
	kill $$LOOKOUT_PID; \
	kill $$DUMMY_PID; \
	exit 1; \

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
