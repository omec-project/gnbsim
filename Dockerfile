# Copyright 2021-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.14.4-stretch AS sim

LABEL maintainer="ONF <omec-dev@opennetworking.org>"

RUN apt-get update
RUN apt-get -y install vim 
RUN apt-get -y install ethtool 
RUN cd $GOPATH/src && mkdir -p gnbsim
COPY . $GOPATH/src/gnbsim 
RUN cd $GOPATH/src/gnbsim && go build -mod=vendor

FROM sim AS gnbsim
RUN mkdir -p /gnbsim/bin
COPY --from=sim $GOPATH/src/gnbsim/gnbsim /gnbsim/bin/
WORKDIR /gnbsim/bin
