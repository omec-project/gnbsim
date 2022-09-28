<!--
SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd

SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0

-->

This repository is part of the SD-Core project. It provides a tool to simulate
gNodeB and UE by generating NAS and NGAP messages for the configured UEs and 
call flows. The tool currently supports simulation profiles for the following
procedures:

    1. Registration
    2. UE Initiated PDU Session Establishment
    3. UE Initiated De-registration 
    4. AN Release
    5. UE Initiated Service Request 
    6. N/W triggered PDU Session Release
    7. UE Requested PDU Session Release
    8. N/W triggered UE Deregistration

It is also capable to generate and send user data packets (ICMP echo request) 
and process downlink user data (ICMP echo response) over the established data 
plane path (N3 Tunnel). 


System level features

    1. Executing all enabled profiles in parallel or in sequential order.
    2. Timeout for each call flow within profile
    3. Logging summary result
    4. HTTP API to execute profile
    5. Configure number of data packets to be sent
    6. Configure AS (Application Server) address. This is used to send data packets
    7. Run gNBSim with single Interface or multi interface


Please refer to the official [SD-Core documentation](https://docs.sd-core.opennetworking.org/master/developer/gnbsim.html#gnb-simulator) for more details. 

## Reach out to us thorugh

1. #sdcore-dev channel in [ONF Community Slack](https://onf-community.slack.com/)
2. Raise Github issues


## gNodeB Simulator Block Diagram

![gNBSim](/docs/images/gnbsim_flow_diagram.png)


## Step 1: Configure gNBSim
    
    1. The config file for gNBSim can be found at <repo dir>/config/gnbsim.yaml
        
        Note: The configuration has following major fields (Read the comments in
        the config file for more details)
            
            - gnbs: 
                List of gNB's to be simulated. Each item in the list holds 
                configuration specific to a gNB.
            - profiles:
                List of test/simulation profiles. Each item in the list holds 
                configuration specific to a profile. Each profile executes   
                required set of systems procedures for the configured set of 
                IMSI's 
        
    2. Enable or disable a specific profile using the "enable" field. 
        
        Currently following profiles are supported :
            - register: 
                Registration procedure
            - pdusessest (Default configured): 
                Registration + UE initiated PDU Session Establishment + User Data
                 packets
            - deregister:
                Registration + UE initiated PDU Session Establishment + User Data
                packets + Deregister
            - anrelease:
                Registration + UE initiated PDU Session Establishment + User Data
                packets + AN Release
            - uetriggservicereq:
                Registration + UE initiated PDU Session Establishment + User Data
                packets + AN Release + UE Initiated Service Request

      
## Step 2: Build gNBSim
    1. Build gNBSim

        $ go build
    
    2.  Build a docker image for gNBSim
        
        $ make docker-build
      
      
## Step 3: Run gNBSim
    
    If you want to run gNBSim as a standalone tool then deploy gNBSim using helm charts. If you want to run gNBSim along with 
    other SD-Core Network Functions then use AIAB to deploy all network functions including gNBSim. 
    
    1. Clone AIAB
    2. Run "make 5g-core"
    3. Trigger call flow testing using following commands
    
    
    Enter gnbsim pod using kubectl exec command and run following commands, 
    
    $ ./gnbsim
    
    Note: By default, the gNB Sim reads the configuration from 
    /gnbsim/config/gnb.conf file. To provide a different configuration file,
    use the below command

    $ ./gnbsim --cfg config/gnbsim.yaml

All these steps are explained in detail on [AIAB documentation](https://docs.sd-core.opennetworking.org/master/developer/aiab.html)

## Step 4: Optionally launching profiles through HTTP APIs

    gNBSim can process HTTP Requests to launch profiles. For example running the
    below curl command will launch a profile in gNBSim
   
    $ curl -i -X POST 127.0.0.1:6000/gnbsim/v1/executeProfile -H 'Content-Type: application/json' -d '{"profileType":"nwreqpdusessrelease","profileName":"profile8","enable":true,"gnbName":"gnb1","startImsi":"208930100007497","ueCount":1,"opc":"981d464c7c52eb6e5036234984ad0bcf","key":"5122250214c33e723a5dd523fc145fc0","sequenceNumber":"16f3b3f70fc2","defaultAs":"192.168.250.1","plmnId":{"mcc":"208","mnc":"93"}}'
