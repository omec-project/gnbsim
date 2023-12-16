#!/bin/bash

# Copyright 2022-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#

set -xe

while getopts p: flag
do
    case "${flag}" in
        p) PROFILE=${OPTARG};;
    esac
done

echo "Profile Name: $PROFILE";

POD_IP=`kubectl get pod gnbsim-0 -n omec --template '{{.status.podIP}}'`

echo $POD_IP

curl -X POST $POD_IP:6000/gnbsim/v1/$PROFILE/stepProfile
