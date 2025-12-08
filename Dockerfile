# Copyright 2021-present Open Networking Foundation
# Copyright 2024-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.25.5-bookworm@sha256:5117d68695f57faa6c2b3a49a6f3187ec1f66c75d5b080e4360bfe4c1ada398c AS builder

RUN apt-get update && \
    apt-get -y install --no-install-recommends \
    vim \
    ethtool && \
    apt-get clean

WORKDIR $GOPATH/src/gnbsim
COPY . .
RUN make all

FROM alpine:3.23@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 AS gnbsim

LABEL maintainer="Aether SD-Core <dev@lists.aetherproject.org>" \
    description="Aether open source 5G Core Network" \
    version="Stage 3"

ARG DEBUG_TOOLS

RUN apk update && apk add --no-cache -U bash tcpdump

# Install debug tools ~ 50MB (if DEBUG_TOOLS is set to true)
RUN if [ "$DEBUG_TOOLS" = "true" ]; then \
        apk update && apk add --no-cache -U gcompat vim strace net-tools curl netcat-openbsd bind-tools; \
        fi

WORKDIR /gnbsim

# Copy executable
COPY --from=builder /go/src/gnbsim/bin /usr/local/bin/.
