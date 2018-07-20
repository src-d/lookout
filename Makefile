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

# SDK
DUMMY_BIN := $(BIN_PATH)/dummy
LOOKOUT_BIN := $(BIN_PATH)/lookout

.PHONY: protogen
protogen:
	$(GOCMD) install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofaster
	protoc \
		-I sdk \
		--gogofaster_out=plugins=grpc,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:pb \
sdk/*.proto

# Integration test for sdk client
.PHONY: test-sdk
test-sdk: clean-sdk build-sdk
	$(DUMMY_BIN) serve &>/dev/null & \
	PID=$$!; \
	$(LOOKOUT_BIN) review ipv4://localhost:10302 | grep "BEGIN RESULT"; \
	if [ $$? != 0 ] ; then \
		echo "review test failed"; \
		kill $$PID; \
		exit 1; \
	fi; \
	$(LOOKOUT_BIN) push ipv4://localhost:10302 | grep "dummy comment for push event"; \
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

clean-sdk:
	rm -f $(DUMMY_BIN)
	rm -f $(LOOKOUT_BIN)
