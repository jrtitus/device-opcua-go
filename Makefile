.PHONY: build test unittest lint clean docker run

MICROSERVICES=cmd/device-opcua

ARCH=$(shell uname -m)

.PHONY: $(MICROSERVICES)

DOCKERS=docker_device_opcua_go

.PHONY: $(DOCKERS)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

GOBIN=$$(go env GOPATH)/bin
GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-opcua-go.Version=$(VERSION)" -trimpath -mod=readonly

tidy:
	go mod tidy

build: $(MICROSERVICES)

build-nats:
	make -e ADD_BUILD_TAGS=include_nats_messaging build

cmd/device-opcua:
	CGO_ENABLED=0 go build -tags "$(ADD_BUILD_TAGS)" $(GOFLAGS) -o $@ ./cmd

unittest:
	go test ./... -coverprofile=coverage.out ./...

install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.63.4

lint:
	@if [ -f $(GOBIN)/golangci-lint ] ; then $(GOBIN)/golangci-lint run --config .golangci.yml ; else echo "WARNING: go linter not installed. To install, run make install-lint"; fi

test: unittest lint
	go vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution-txt.sh

clean:
	rm -f $(MICROSERVICES)

run:
	cd bin && ./edgex-launch.sh

docker: $(DOCKERS)

docker_device_opcua_go:
	docker buildx build --pull \
		--label "git_sha=$(GIT_SHA)" \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		-t edgexfoundry/device-opcua-go:$(VERSION)-dev \
		--load .

docker-nats:
	make -e ADD_BUILD_TAGS=include_nats_messaging docker

vendor:
	CGO_ENABLED=0 go mod vendor
