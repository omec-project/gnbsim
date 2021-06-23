# Copyright 2019-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.14.4-stretch AS gnbsim

LABEL maintainer="ONF <omec-dev@opennetworking.org>"

RUN apt-get update
RUN apt-get -y install vim 
RUN cd $GOPATH/src && mkdir -p gnbsim
COPY . $GOPATH/src/gnbsim 
COPY ./patches/NAS_DLNASTransport.go $GOPATH/src/gnbsim/vendor/github.com/free5gc/nas/nasMessage/
RUN cd $GOPATH/src/gnbsim && go build -mod=vendor
WORKDIR $GOPATH/src/gnbsim
