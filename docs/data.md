<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0

-->

# gNBsim data support


gNBSim can generate and send user data packets (ICMP echo request)
and process downlink user data (ICMP echo response) over the established user
plane path (GTP Tunnel). Configure number of data packets to be sent. Configure
AS (Application Server) address. This is used to send data packets.


      - profileType: nwtriggeruedereg # profile type
        profileName: profile6 # uniquely identifies a profile within application
        enable: false # Set true to execute the profile, false otherwise.
        gnbName: gnb1 # gNB to be used for this profile
        startImsi: 208930100007497 # First IMSI. Subsequent values will be used if ueCount is more than 1
        ueCount: 1 # Number of UEs for for which the profile will be executed
        defaultAs: "192.168.250.1" # default icmp pkt destination
        perUserTimeout: 10 # if no expected event received in this time then treat it as failure
