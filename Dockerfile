# Copyright 2021-present Open Networking Foundation
# Copyright 2024-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.26.5-bookworm@sha256:18aedc16aa19b3fd7ded7245fc14b109e054d65d22ed53c355c899582bbb2113 AS builder

WORKDIR $GOPATH/src/gnbsim
COPY . .
ARG MAKEFLAGS
RUN make all

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS gnbsim

# Build arguments for dynamic labels
ARG VERSION=dev
ARG VCS_URL=unknown
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.source="${VCS_URL}" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.revision="${VCS_REF}" \
    org.opencontainers.image.url="${VCS_URL}" \
    org.opencontainers.image.title="gnbsim" \
    org.opencontainers.image.description="Aether 5G Core GNBSIM Network Function" \
    org.opencontainers.image.authors="Aether SD-Core <dev@lists.aetherproject.org>" \
    org.opencontainers.image.vendor="Aether Project" \
    org.opencontainers.image.licenses="Apache-2.0" \
    org.opencontainers.image.documentation="https://docs.sd-core.aetherproject.org/"

ARG DEBUG_TOOLS

RUN apk add --no-cache bash tcpdump && \
    if [ "$DEBUG_TOOLS" = "true" ]; then \
    apk add --no-cache gcompat vim strace net-tools curl netcat-openbsd bind-tools; \
    fi

WORKDIR /gnbsim

# Copy executable
COPY --from=builder /go/src/gnbsim/bin/* /usr/local/bin/.
