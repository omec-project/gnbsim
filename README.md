<!--
SPDX-FileCopyrightText: 2024 Intel Corporation
SPDX-FileCopyrightText: 2022 Great Software Laboratory Pvt. Ltd
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
-->
[![Go Report Card](https://goreportcard.com/badge/github.com/omec-project/gnbsim)](https://goreportcard.com/report/github.com/omec-project/gnbsim)

# Table Of Contents
* [Introduction](#Introduction)
* [gNBSim Block Diagram](#gnbsim-simulator-block-diagram)
* [Supported Features](#supported-features)
* [Pending Features](#pending-features)
* [Support & Contributions](#Support-and-contributions)
* [License](#license)
* [Deployment model](/docs/deployment.md)
* [Configuration](/docs/config.md)
* Advanced Configuration/Features
  * [API Based Configuration](/docs/apis.md)
  * [Data Support](/docs/data.md)
  * [Message Flow](/docs/gnbsim_flow.md)
  * [Profile Details](/docs/profile_manager_design.md)
  * [UE Originated Message Flow](/docs/ue_originated_flow.md)
  * [UE Terminated Message Flow](/docs/ue_terminated_flow.md)


# Introduction

This repository is part of the SD-Core project. SD-Core is open source 5G
implementation of 5G Core Network. SD-Core provides a tool to simulate gNodeB
and UE by generating NAS and NGAP messages for the configured UEs and call flows.

# gNBSim Simulator Block Diagram

![gNBSim](/docs/images/gnbsim_flow_diagram.png)


# Supported features

Supported 3gpp procedures:

- UE Registration
- UE Initiated PDU Session Establishment
- UE Initiated De-registration
- AN Release
- UE Initiated Service Request
- N/W triggered PDU Session Release
- UE Requested PDU Session Release
- N/W triggered UE Deregistration

Supported System level features

- Gnbsim can generate and send user data packets (ICMP echo request) and process
downlink user data (ICMP echo response) over the established data plane path
(N3 Tunnel)
- Executing all enabled profiles in parallel or in sequential order
- Timeout for each call flow within profile
- Logging summary result
- HTTP API to execute profile
- Configure number of data packets to be sent and time interval between consecutive packets
- Configure AS (Application Server) address. This is used to send data packets
- Run gNBSim with single Interface or multi interface
- Support of Custom Profiles
- Delay between Procedures
- Timeout for every profile
- Logic to calculate latency per transaction/ operation
- Support retransmission of Service Request Message
- Support deployment of gNBSim as standalone container
- Reporting profile errors from all levels

# Pending Features

- Provision data interface to gNBSim Container/POD/executable for data traffic testing
- [3gpp] Generating GTPU echo request & handle incoming GTPU response
- [3gpp] Handle incoming GTPu echo request & generate GTPu echo response
- [3gpp] Support to send Error indication Message
- [3gpp] Adding support for handling End Marker packet
- [3gpp] GUTI based registration
- [3gpp] Adding support for Resynchronization Profile
- [3gpp] Adding Support for N2 handover profile
- Support to handle Paging Request
- Controlling Profiles - Adding support for aborting profile
- Controlling Profiles - Suspend & resume  profiles
- Adding support for configurable rate of events
- HTTP APIs to fetch subscriber/profile status from gNBSim
- Adding Support for Xn Handover profile
- Dropping incoming messages based on configuration
- Sending negative responses to request/command type messages based on configuration
- Handling security mode failure message
- Triggering downlink data from gNB Sim (CI/CD feature as well)
- UI Support
- 3gpp Release Upgrade
- Extensive gnbsim documentation
- Command line options to send logs or events to file in specific format for post processing.

# Support and Contributions

The gnbsim project welcomes new contributors. Feel free to propose a new feature or fix bugs!

Before contributing, please follow these guidelines:

1. gNBSim documentation details [here](./docs/README.md)
2. Please refer to the official [SD-Core documentation](https://docs.sd-core.opennetworking.org/master/developer/gnbsim.html#gnb-simulator) for more details.
3. #sdcore-dev channel in [ONF Community Slack](https://onf-community.slack.com/)
4. Raise Github issues

# License

The project is licensed under the [Apache License, version 2.0](./LICENSES/Apache-2.0.txt).
