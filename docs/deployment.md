<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0

-->

# gNBSim deployment options

## As standalone docker container

TBD

## As standalone executable

TBD

## As kubernetes pod

TBD

## As a part of Aether In a Box

- To quickly launch and test AiaB with 5G SD-CORE using gNBSim:

        $ make 5g-test

- Alternatively, you can do following
        $ make 5g-core
        # Once all PODs are up then you can enter into the gNBSim pod by running
        $ kubectl exec -it gnbsim-0 -n omec bash
        $ ./gnbsim
        # By default, the gNB Sim reads the configuration from  /gnbsim/config/gnb.conf file.
        # To provide a different configuration file, use the below command
        $ ./gnbsim --cfg <config file path>
 
