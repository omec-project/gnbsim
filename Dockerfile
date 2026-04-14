# Copyright 2021-present Open Networking Foundation
# Copyright 2024-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.26.2-bookworm@sha256:4f4ab2c90005e7e63cb631f0b4427f05422f241622ee3ec4727cc5febbf83e34 AS builder

WORKDIR $GOPATH/src/gnbsim
COPY . .
RUN make all

FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659 AS gnbsim

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

RUN apk update && apk add --no-cache bash tcpdump

# Install debug tools only when explicitly requested.
RUN if [ "$DEBUG_TOOLS" = "true" ]; then \
    apk add --no-cache gcompat vim strace net-tools curl netcat-openbsd bind-tools; \
    fi

WORKDIR /gnbsim

# Copy executable
COPY --from=builder /go/src/gnbsim/bin /usr/local/bin/.
