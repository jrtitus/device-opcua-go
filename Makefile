.PHONY: build clean_build clean_tests docker run test

GO=GO111MODULE=on go
GOCGO=CGO_ENABLED=1 $(GO)

MICROSERVICES=cmd/device-opcua

.PHONY: $(MICROSERVICES)

VERSION=$(shell cat ./VERSION)
GIT_SHA=$(shell git rev-parse HEAD)
CURR_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-opcua-go.Version=$(VERSION)"
TEST_ARTIFACTS=test-artifacts

build: $(MICROSERVICES)
	$(GOCGO) install -tags=safe

cmd/device-opcua: clean_build
	$(GOCGO) build $(GOFLAGS) -o $@ ./cmd

clean_build:
	rm -f $(MICROSERVICES)

clean_tests:
	rm -rf $(TEST_ARTIFACTS)

docker: clean_build
	DOCKER_BUILDKIT=1 docker build \
		--label "git_sha=$(GIT_SHA)" \
		-t edgexfoundry/device-opcua-go:$(VERSION) \
		--target prod \
		--pull \
		.

run:
	cd bin && ./edgex-launch.sh

test: clean_tests
	mkdir -p $(TEST_ARTIFACTS)
	DOCKER_BUILDKIT=1 docker build \
		-t device-opcua-go:$(VERSION)-test \
		--target tester \
		--build-arg TEST_ARTIFACTS=$(TEST_ARTIFACTS) \
		.
	docker run --rm -v $(CURR_DIR)/$(TEST_ARTIFACTS):/device-opcua-go/$(TEST_ARTIFACTS) device-opcua-go:$(VERSION)-test
