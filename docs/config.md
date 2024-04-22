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
    Support of multiple gNBs: Two gNBs are configured by default. So, User can create profiles by using these gNBs.
    Configuration of two gNBs can be found here

        gnb:
          ips:
          - '"192.168.251.5/24"' # gnb1 IP
          - '"192.168.251.6/32"' # gnb2 IP
        configuration:
          runConfigProfilesAtStart: true
          singleInterface: # this will be added through configmap script
          execInParallel: false # run all profiles in parallel
          gnbs: # pool of gNodeBs
            gnb1:
              n2IpAddr: # gNB N2 interface IP address used to connect to AMF
              n2Port: 9487 # gNB N2 Port used to connect to AMF
              n3IpAddr: 192.168.251.5 # gNB N3 interface IP address used to connect to UPF
              n3Port: 2152 # gNB N3 Port used to connect to UPF
              name: gnb1 # gNB name that uniquely identifies a gNB within application
              globalRanId:
                plmnId:
                  mcc: 208 # Mobile Country Code (3 digits string, digit: 0~9)
                  mnc: 93 # Mobile Network Code (2 or 3 digits string, digit: 0~9)
                gNbId:
                  bitLength: 24
                  gNBValue: "000102" # gNB identifier (3 bytes hex string, range: 000000~FFFFFF)
              supportedTaList:
              - tac: "000001" # Tracking Area Code (3 bytes hex string, range: 000000~FFFFFF)
                broadcastPlmnList:
                  - plmnId:
                      mcc: 208
                      mnc: 93
                    taiSliceSupportList:
                        - sst: 1 # Slice/Service Type (uinteger, range: 0~255)
                          sd: "010203" # Slice Differentiator (3 bytes hex string, range: 000000~FFFFFF)
              defaultAmf:
                hostName: amf # Host name of AMF
                ipAddr: # AMF IP address
                port: 38412 # AMF port
            gnb2:
              n2IpAddr: # gNB N2 interface IP address used to connect to AMF
              n2Port: 9488 # gNB N2 Port used to connect to AMF
              n3IpAddr: 192.168.251.6 # gNB N3 interface IP address used to connect to UPF
              n3Port: 2152 # gNB N3 Port used to connect to UPF
              name: gnb2 # gNB name that uniquely identify a gNB within application
              globalRanId:
                plmnId:
                  mcc: 208 # Mobile Country Code (3 digits string, digit: 0~9)
                  mnc: 93 # Mobile Network Code (2 or 3 digits string, digit: 0~9)
                gNbId:
                  bitLength: 24
                  gNBValue: "000112" # gNB identifier (3 bytes hex string, range: 000000~FFFFFF)
              supportedTaList:
              - tac: "000001" # Tracking Area Code (3 bytes hex string, range: 000000~FFFFFF)
                broadcastPlmnList:
                  - plmnId:
                      mcc: 208
                      mnc: 93
                    taiSliceSupportList:
                        - sst: 1 # Slice/Service Type (uinteger, range: 0~255)
                          sd: "010203" # Slice Differentiator (3 bytes hex string, range: 000000~FFFFFF)
              defaultAmf:
                hostName: amf # Host name of AMF
                ipAddr: # AMF IP address
                port: 38412 # AMF port

- **profiles:**
	List of test/simulation profiles. Each item in the list holds configuration specific to a profile.
	Each profile executes required set of systems procedures for the configured set of IMSI's.
	Each profile has enable field and if set to true then that profile is executed.

- **Getting gNBSim golang profile**

       config:
         gnbsim:
           goProfile:
             enable: true #enable/disable golang profile in gnbsim
             port: 5000        

- **Run gNBSim with single Interface or multi interface**

       config:
         gnbsim:
           yamlCfgFiles:
             gnb.conf:
               configuration:
                   singleInterface: false #default false i.e. multiInterface. Works well for AIAB

- **CustomProfiles**
    List of custom profiles. Each item in the list holds configuration specific to a customProfile.
    Each custom profile has enable field and if set to true then that custom profile is executed.
    Support of Custom Profiles: User can now define your own profile. New profile can be
    created by using existing baseline procedure. Example of custom profile can be found here.
    Check customProfiles in `gNBSim config <https://github.com/omec-project/gnbsim/blob/main/config/gnbsim.yaml>_`
    Delay between Procedures can be added using customProfiles

       customProfiles:
         customProfiles1:
           profileType: custom # profile type
           profileName: custom1 # uniqely identifies a profile within application
           enable: false # Set true to execute the profile, false otherwise.
           execInParallel: false # run all subscribers in parallel
           stepTrigger: true # wait for trigger to move to next step
           gnbName: gnb1 # gNB to be used for this profile
           startImsi: 208930100007487
           ueCount: 5
           defaultAs: "192.168.250.1" #default icmp pkt destination
           opc: "981d464c7c52eb6e5036234984ad0bcf"
           key: "5122250214c33e723a5dd523fc145fc0"
           sequenceNumber: "16f3b3f70fc2"
           plmnId: # Public Land Mobile Network ID, <PLMN ID> = <MCC><MNC>
             mcc: 208 # Mobile Country Code (3 digits string, digit: 0~9)
             mnc: 93 # Mobile Network Code (2 or 3 digits string, digit: 0~9)
           startiteration: iteration1
           iterations:
             #at max 7 actions
             - "name": "iteration1"
               "1": "REGISTRATION-PROCEDURE 5"
               "2": "PDU-SESSION-ESTABLISHMENT-PROCEDURE 5"  #5 second delay after this procedure
               "3": "USER-DATA-PACKET-GENERATION-PROCEDURE 10"
               "next":  "iteration2"
             - "name": "iteration2"
               "1": "AN-RELEASE-PROCEDURE 100"
               "2": "UE-TRIGGERED-SERVICE-REQUEST-PROCEDURE 10"
               "repeat": 5
               "next":  "iteration3"
             - "name": "iteration3"
               "1": "UE-INITIATED-DEREGISTRATION-PROCEDURE 10"
               #"repeat": 0 #default value 0 . i.e execute once
               #"next":  "quit" #default value quit. i.e. no further iteration to run

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

