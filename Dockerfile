# Copyright 2021-present Open Networking Foundation
# Copyright 2024-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.25.4-bookworm@sha256:e17419604b6d1f9bc245694425f0ec9b1b53685c80850900a376fb10cb0f70cb AS builder

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
