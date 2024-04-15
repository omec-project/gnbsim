<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0

-->

# Configure gNBSim
    
## Config file
        
>[!NOTE]
> The configuration for gNBSim can be found [here](../config/gnbsim.yaml)
            
- **gnbs:** 
    List of gNB's to be simulated. Each item in the list holds configuration specific to a gNB.

- **profiles:**
	List of test/simulation profiles. Each item in the list holds configuration specific to a profile.
	Each profile executes required set of systems procedures for the configured set of IMSI's.
	Each profile has enable field and if set to true then that profile is executed.
        
## Description of Each Profile
        
Currently following profiles are supported :

### **register:**
- Registration procedure

### **pdusessest (Default configured):**
- Registration + UE initiated PDU Session Establishment + User Data packets

### **deregister:**
- Registration + UE initiated PDU Session Establishment + User Data packets + Deregister

### **anrelease:**
- Registration + UE initiated PDU Session Establishment + User Data packets + AN Release

### **uetriggservicereq:**
- Registration + UE initiated PDU Session Establishment + User Data packets + AN Release + UE Initiated Service Request
