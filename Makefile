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

protogen:
	$(GOCMD) install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofaster
	protoc \
		-I pb -I vendor -I vendor/github.com/gogo/protobuf/protobuf \
		--gogofaster_out=plugins=grpc:pb pb/*.proto
