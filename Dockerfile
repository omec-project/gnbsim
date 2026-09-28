# Copyright 2021-present Open Networking Foundation
# Copyright 2024-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS builder

WORKDIR $GOPATH/src/gnbsim
COPY . .
ARG MAKEFLAGS
RUN make all

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS gnbsim

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
