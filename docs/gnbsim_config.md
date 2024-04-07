<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0

-->

# Configure gNBSim
    
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



