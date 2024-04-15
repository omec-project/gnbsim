<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0

-->

## gNBsim can Optionally launch profiles through HTTP APIs

gNBSim can process HTTP Requests to launch profiles. For example running the
below curl command will launch a profile in gNBSim
   
    $ curl -i -X POST 127.0.0.1:6000/gnbsim/v1/executeProfile -H 'Content-Type: application/json' -d '{"profileType":"nwreqpdusessrelease","profileName":"profile8","enable":true,"gnbName":"gnb1","startImsi":"208930100007497","ueCount":1,"opc":"981d464c7c52eb6e5036234984ad0bcf","key":"5122250214c33e723a5dd523fc145fc0","sequenceNumber":"16f3b3f70fc2","defaultAs":"192.168.250.1","plmnId":{"mcc":"208","mnc":"93"}}'

