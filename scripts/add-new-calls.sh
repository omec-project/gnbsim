#!/bin/bash

# Copyright 2022-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#
# Add new calls in existing profile
# ./add-new-calls.sh -p profile1 -n 10


set -xe

while getopts p:n: flag
do
    case "${flag}" in
        p) PROFILE=${OPTARG};;
        n) NUMBER=${OPTARG};;
    esac
done

echo "Profile Name: $PROFILE";
echo "Number of calls: $NUMBER";

POD_IP=`kubectl get pod gnbsim-0 -n omec --template '{{.status.podIP}}'`
echo $POD_IP

if [ -z "$POD_IP" ]
then
echo "POD IP empty"
return
fi

if [ -z "$NUMBER" ]
then
NUMBER=1
fi

curl -X POST $POD_IP:6000/gnbsim/v1/$PROFILE/addNewCalls?number=$NUMBER
