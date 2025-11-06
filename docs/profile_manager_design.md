<!--
SPDX-FileCopyrightText: 2022  Great Software Laboratory Pvt. Ltd

SPDX-License-Identifier: Apache-2.0

-->

# Profile Manager Design documentation

* Executing all enabled profiles in parallel or in sequential order.

       config:
         gnbsim:
           yamlCfgFiles:
             gnb.conf:
               configuration:
                   execInParallel: false #run all profiles in parallel

> [!NOTE]
> There is execInParallel option under each profile as well. execInParallel under profile means that all the
> subscribers in the profile are run in parallel

* Timeout for each call flow within profile

       - profileType: nwtriggeruedereg # profile type
         profileName: profile6 # uniqely identifies a profile within application
         perUserTimeout: 10 #if no expected event received in this time then treat it as failure

## Profile Manager Overview

![gNBSim](/docs/images/profile_manager_overview.png)

## Profile Manager Initialization Sequence

![gNBSim](/docs/images/profile_manager_initialization_seq_diag.png)
