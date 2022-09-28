#!/bin/bash

# Copyright 2022-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#
# Create new profile 
# ./create-new-profile.sh -p profilenew -n 10


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

curl -X POST $POD_IP:6000/gnbsim/v1/$PROFILE/addNewCalls?num=$NUMBER

curl -i -X POST $POD_IP:6000/gnbsim/v1/executeProfile -H 'Content-Type: application/json' -d '{"profileType":"register","profileName":"'$PROFILE'","enable":true,"gnbName":"gnb1","startImsi":"208930100007497","ueCount":'$NUMBER',"opc":"981d464c7c52eb6e5036234984ad0bcf","key":"5122250214c33e723a5dd523fc145fc0","sequenceNumber":"16f3b3f70fc2","defaultAs":"192.168.250.1","plmnId":{"mcc":"208","mnc":"93"}}'
