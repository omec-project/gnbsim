<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0

-->

# Pending Feature List

## Common features for gNodeB Simulator

    1. Adding support for custom profile
    2. Controlling Profiles - Adding support for aborting profile
    3. Controlling Profiles - Clean profiles
    4. Controlling Profiles - Pause Profile
    5. Adding support for configurable rate of events
    
 ## Data Testing Features
    1. Triggering downlink data from gNB Sim (CI/CD feature as well) 
    2. Provision data interface to gNBSim POD for data traffic testing
    
 ## CI/cd features
 
    1. Adding configurable delay between profile execution
    2. Advanced logging - Logic to calculate latency per transaction/ operation
    3. Profile execution through http APIs (partially supported)
    4. Reporting profile errors from all levels 
    4. HTTP APIs to fetch subscriber/profile status from gNBSim

## Negative Testing features

    1. Dropping incoming messages based on configuration
    2. Sending negative responses to request/command type messages based on configuration
    3. Handling security mode failure message
    
 ## 3gpp features for gNodeB Simulator
 
    1. GUTI based registration 
    2. Adding support for Resynchronization Profile
    3. Adding Support for N2 handover profile
    4. Adding Support for Xn Handover profile
    5. Adding support for handling end marker packet
    6. Generating GTPU echo request and handling incoming GTPU Request
