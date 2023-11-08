# Copyright 2021-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.21.3-bookworm AS gnb

LABEL maintainer="ONF <omec-dev@opennetworking.org>"

RUN apt-get update && apt-get -y install vim ethtool
RUN cd $GOPATH/src && mkdir -p gnbsim
COPY . $GOPATH/src/gnbsim 
RUN cd $GOPATH/src/gnbsim && go build -mod=vendor

FROM alpine:3.16 AS gnbsim
RUN apk update && apk add -U gcompat vim strace net-tools curl netcat-openbsd bind-tools bash tcpdump

RUN mkdir -p /gnbsim/bin

# Copy executable
COPY --from=gnb /go/src/gnbsim/gnbsim /gnbsim/bin/
WORKDIR /gnbsim/bin
