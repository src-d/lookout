# Package configuration
PROJECT = lookout
COMMANDS = cmd/lookoutd
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
LOOKOUT_SDK_BIN := $(BIN_PATH)/lookout-sdk

# lookoutd binary
LOOKOUT_BIN := $(BIN_PATH)/lookoutd

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

GOTEST_INTEGRATION = $(GOTEST) -tags=integration

# Integration test for sdk client
.PHONY: test-sdk
test-sdk: clean-all build-all
	DUMMY_BIN=$(PWD)/$(DUMMY_BIN) \
	LOOKOUT_BIN=$(PWD)/$(LOOKOUT_SDK_BIN) \
	$(GOTEST_INTEGRATION) github.com/src-d/lookout/cmd/sdk-test

# Same as test-sdk, but skipping tests that require a bblfshd server
.PHONY: test-sdk-short
test-sdk-short: clean-all build-all
	DUMMY_BIN=$(PWD)/$(DUMMY_BIN) \
	LOOKOUT_BIN=$(PWD)/$(LOOKOUT_SDK_BIN) \
	$(GOTEST_INTEGRATION) -test.short github.com/src-d/lookout/cmd/sdk-test

# Integration test for lookout serve
.PHONY: test-json
test-json: clean build-all
	DUMMY_BIN=$(PWD)/$(DUMMY_BIN) \
	LOOKOUT_BIN=$(PWD)/$(LOOKOUT_BIN) \
	$(GOTEST_INTEGRATION) github.com/src-d/lookout/cmd/server-test

# Build sdk client and dummy analyzer
.PHONY: build-all
build-all: $(DUMMY_BIN) $(LOOKOUT_BIN) $(LOOKOUT_SDK_BIN)
$(LOOKOUT_BIN):
	$(GOBUILD) -o "$(LOOKOUT_BIN)" ./cmd/lookoutd
$(LOOKOUT_SDK_BIN):
	$(GOBUILD) -o "$(LOOKOUT_SDK_BIN)" ./cmd/lookout-sdk
$(DUMMY_BIN):
	$(GOBUILD) -o "$(DUMMY_BIN)" ./cmd/dummy

.PHONY: clean-all
clean-all:
	rm -f $(DUMMY_BIN)
	rm -f $(LOOKOUT_BIN)
	rm -f $(LOOKOUT_SDK_BIN)

.PHONY: dry-run
dry-run: $(CONFIG_FILE)
	go run cmd/lookoutd/*.go serve --dry-run github.com/src-d/lookout
$(CONFIG_FILE):
	cp "$(CONFIG_FILE).tpl" $(CONFIG_FILE)

# TODO: remove when https://github.com/src-d/ci/pull/84 is merged
.PHONY: godep
GODEP ?= $(CI_PATH)/dep
godep:
	export INSTALL_DIRECTORY=$(CI_PATH) ; \
	test -f $(GODEP) || \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | bash ; \
	$(GODEP) ensure -v
