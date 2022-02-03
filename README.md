<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0

-->

## How to Use
1. Clone gnbSim
2. Create image - sudo make docker-build
3. Use newly created image in the aiab oerride file (make 5gc)

## How to run Testcases 

1. Once gnbsim is started, get into gnbsim pod 
    kubectl exec -it gnbsim-0 -n omec bash
2. cd src/gnbsim
3. Now you can run multiple testcases in this running container

    Register Testcase    - ./gnbsim register  
    
    DeRegister Testcase    - ./gnbsim deregister  
    
    xnhandover Testcase - ./gnbsim xnhandover 

    n2handover Testcase - ./gnbsim n2handover 

## Code Change in running container
If any code is changed within container then use following command 
root@gnbsim-0:/go/src/gnbsim# go build gnbsim.go
