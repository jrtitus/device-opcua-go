#
# Copyright (c) 2018, 2019 Intel
# Copyright (c) 2021-2023 Schneider Electric
#
# SPDX-License-Identifier: Apache-2.0
#
FROM golang:1.20-alpine3.18 AS builder
WORKDIR /device-opcua-go

# Install our build time packages.
RUN apk update && apk add --no-cache make git gcc pkgconfig musl-dev

COPY go.* ./

RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

COPY . .

ARG ADD_BUILD_TAGS=""
RUN make -e ADD_BUILD_TAGS=$ADD_BUILD_TAGS build

# Next image - Copy built Go binary into new workspace
FROM alpine:3.18

# dumb-init needed for injected secure bootstrapping entrypoint script when run in secure mode.
RUN apk add --update --no-cache dumb-init

# expose command data port
EXPOSE 59997

COPY --from=builder /device-opcua-go/cmd/device-opcua /
COPY --from=builder /device-opcua-go/cmd/res /res
COPY LICENSE /

ENTRYPOINT ["/device-opcua"]
CMD ["--cp=consul://edgex-core-consul:8500", "--registry"]
