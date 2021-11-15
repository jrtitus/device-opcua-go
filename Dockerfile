#
# Copyright (c) 2018, 2019 Intel
# Copyright (c) 2021 Schneider Electric
#
# SPDX-License-Identifier: Apache-2.0
#
FROM golang:1.16-alpine3.14 AS base
WORKDIR /device-opcua-go

# Install our build time packages.
RUN apk add --update --no-cache make git zeromq-dev gcc pkgconfig musl-dev

ADD cmd ./cmd
ADD internal ./internal
COPY go.* VERSION version.go ./

FROM base AS builder
WORKDIR /device-opcua-go

COPY Makefile .

RUN make build

# Production image - Copy built Go binary into new workspace
FROM alpine:3.14 AS prod

# dumb-init needed for injected secure bootstrapping entrypoint script when run in secure mode.
RUN apk add --update --no-cache zeromq dumb-init

# expose command data port
EXPOSE 59997

COPY --from=builder /device-opcua-go/cmd/device-opcua /
COPY --from=builder /device-opcua-go/cmd/res /res
COPY LICENSE /
COPY Attribution.txt /

ENTRYPOINT ["/device-opcua"]
CMD ["--cp=consul://edgex-core-consul:8500", "--registry", "--confdir=/res"]

FROM base AS test_base

# Install packages required to run mock opcua server
RUN apk add --update --no-cache bash python3 py3-pip py3-wheel libxml2-dev libxslt-dev python3-dev && \
    python3 -m pip install opcua

# Install test packages for generating coverage metrics
RUN go install github.com/jstemmer/go-junit-report@v0.9.1 && \
    go install github.com/axw/gocov/gocov@v1.0.0 && \
    go install github.com/AlekSi/gocov-xml@v1.0.0 && \
    go install github.com/jandelgado/gcov2lcov@v1.0.5

FROM test_base AS tester
ARG TEST_ARTIFACTS=test-artifacts

WORKDIR /device-opcua-go

ADD bin ./bin
COPY test.sh .
COPY Attribution.txt .
RUN chmod +x test.sh ./bin/test-attribution-txt.sh

ENV TEST_ARTIFACTS=${TEST_ARTIFACTS}
CMD /device-opcua-go/test.sh ${TEST_ARTIFACTS}
