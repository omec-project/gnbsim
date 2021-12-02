<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0
-->

The GNBSIM tool simulates gNodeB and UE by generating NAS and NGAP messages for 
the configured UEs and call flows. It currently supports Registration, UE 
initiated PDU Session Establishment, UE Initiated Deregistration procedures and 
is capable to generate and send a user data packet (ICMP echo request) and 
process downlink user data (ICMP echo response) over the established data plane 
path (N3 Tunnel). 
To simulate other call flows, kindly use the following docker image:
    ajaythakuronf/5gc-gnbsim:0.0.9-dev

## Step 1: Configure GNBSIM
    
    1. The config file for GNBSIM can be found at <repo dir>/config/gnbsim.yaml
        
        Note: The configuration has following major fields:
            - gnbs: 
                List of gNB's to be simulated. Each item in the list holds 
                configuration specific to a gNB.
            - profiles:
                List of test/simulation profiles. Each item in the list holds 
                configuration specific to a profile.
    
        Read the comments in the config file for more details    
        
    2. Enable or disable a specific profile using the "enable" field. 
        
        Currently following profiles are supported :
            - pdusessest : Registration + UE initiated PDU Session Establishment + User Data packets
            - register   : Registration procedure
   
        Note: The default configuration has the "pdusessest" type profile enabled
      
## Step 2: Build GNBSIM
    1. To modify GNBSIM within a container run 
        
        $ kubectl exec -it gnbsim-0 -n omec bash
        make required changes and run
        $ go build
    
    2.  To modify GNBSIM and build a new docker image
        
        $ cd <repo dir>
        $ make docker-build
      
        To use newly created image in the AIAB cluster run, 
        $ cd <aiab repo dir>
        $ make reset-5g-test
        modify the override file (ransim-values.yaml) to add the new image name
        $ make 5gc
      
      
## Step 3: Run GNBSIM
   
    Once GNBSIM is started, get into GNBSIM pod by running
    $ kubectl exec -it gnbsim-0 -n omec bash
    After entering the pod run,
    
    $ ./gnbsim
    
    Note: By default, the gNB Sim reads the configuration from 
    /free5gc/config/gnb.conf file. To provide a different configuration file,
    use the below command

    $ ./gnbsim --cfg config/gnbsim.yaml
