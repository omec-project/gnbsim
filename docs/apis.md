<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0

-->

## gNBsim can Optionally control profiles through HTTP APIs

gNBSim can process HTTP Requests to launch profiles. 

HTTP API to create new profile. Below configuration enables http server in gNBSim.


      config:
        gnbsim:
          httpServer:
            enable: true #enable httpServer in gnbsim
            port: 6000


## gNBsim can Optionally launch profiles through HTTP APIs

Refer [create new profile script](/scripts/create-new-profile.sh)
   
## gNBsim can Optionally add new calls to existing profiles through HTTP APIs

Refer [add new calls scripts](/scripts/add-new-calls.sh)

## gNBsim can Optionally calls APIs to step through executions through HTTP APIs

All profiles are given event to execute one step in the profile and wait for next
step API call for further profile execution. Please refer following config in the
custom profile to enable stepTrigger.

     customProfiles:
       customProfiles1:
         stepTrigger: true #wait for trigger to move to next step


Refer [step profiles scripts](/scripts/step-profile.sh)
