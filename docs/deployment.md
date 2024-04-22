<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0

-->

# gNBSim deployment options

## Option1: As docker container 

[Aether onRamp](https://docs.aetherproject.org/master/onramp/gnbsim.html) installs gNBSim as docker container next to SD-Core (5G Core).
If user wish to use just gNBSim and test it with some other 5G Core then Aether onRamp allows just installation of gNBSim. 
Aether onRamp makefiles are readable and  user can achieve the required goal.

## Option2: As standalone executable

If you want to run gNBSim as a standalone running executable tool then follow below steps.

        $ git clone git@github.com:omec-project/gnbsim.git
        $ cd gnbsim
        $ go build

Run below command to run 5G call flows testing using default configuration file

        $ ./gnbsim

To provide a different configuration file, use the below command

         $ ./gnbsim --cfg config/gnbsim.yaml

> [!Note] By default, the gNBSim reads the configuration from ./config/gnbsim.yaml file.
> Please refer to the gNBSim configuration [guide](./docs/config.md). Please make sure following
> configuration fields are updated to correct values.

For each gNB in the config file change below values,

      n2IpAddr: # gNB N2 interface IP address used to connect to AMF
      n3IpAddr: 192.168.251.5 # gNB N3 interface IP address used to connect to UPF. when singleInterface mode is false

Update amf endpoint to correct FQDN or IP address

      defaultAmf:
        hostName: amf # Host name of AMF
        ipAddr: # AMF IP address
        port: 38412 # AMF port

Update defaultAs value to correct IP address. This address is used to send some test data packets after UE PDU session establishment.

      defaultAs: "192.168.250.1" #default icmp pkt destination


## Option3: As a part of Aether In a Box

- This is default mode of deployment for gNB Simulator
- Multus cni needs to be enabled on cluster. Required for bess-upf & gNBSim
- `make 5gc` will by default deploy gNB Simulator in this mode
- One interface is used for user plane traffic towards UPF
- Second interface is used to send traffic towards control plane (i.e. AMF).
- UPF network & default gateway is provided in the override values.
- Route to UPF network is added when POD comes up
- defaultAs is configured per profile. This address is used to send data traffic during test


![Aether in a Box](/docs/images/Single-cluster_2_interface.jpg)

To add UPF routes. Following is example of override values ::

     config:
       gnbsim:
         gnb:
           ip: 192.168.251.5/24 #user plane IP at gnb if 2 separate interface provided
         singleInterface: false
         networkTopo:
           - upfAddr: "192.168.252.3/32"
             upfGw: "192.168.251.1"


This is how it is used in [Aether In a Box](https://gerrit.opencord.org/plugins/gitiles/aether-in-a-box/+/refs/heads/master)
This environment is not actively supported. Most of the community developers have moved to Aether onRamp.
Below are the steps to install gNBSim in Aether in a Box (AIAB).


        $ git clone "https://gerrit.opencord.org/aether-in-a-box"
        $ cd aether-in-a-box/

To quickly launch and test AiaB with 5G SD-CORE using gNBSim. Below command will install K8s cluster on the single node.

        $ make 5g-test #to test 5G calls. This also installes 5G Core & gnbSim


To clean the deployment run below command

        $ make reset-5g-test #to uninstall the 5G Core & gNBSim


Alternatively, if you just want to install 5G Core but not to run any tests then do following,

        $ make 5g-core
        # Once all PODs are up then you can enter into the gNBSim pod by running
        $ kubectl exec -it gnbsim-0 -n omec bash
        $ ./gnbsim #this triggers the default profile tests

        # By default, the gNB Sim reads the configuration from  /gnbsim/config/gnb.conf file.
        # To provide a different configuration file, use the below command
        $ ./gnbsim --cfg <config file path>
 
## Option4: As kubernetes pod

Helm charts for gNBSim can be found [here](https://github.com/omec-project/sdcore-helm-charts/tree/main/5g-ran-sim)
gNBSim image can be found in the docker hub [here](https://hub.docker.com/r/omecproject/5gc-gnbsim/tags)

**TBD**: helm chart commands to install gNBSim on any existing kubernetes cluster.

**TBD**: Provide clean steps so that gNBSim can be installed without Aether In a Box need.
 
If you find a need to change gNBSim code and use the updated image in the test setup then follow below steps.
To modify gNBSim and build a new docker image:

     $ git clone git@github.com:omec-project/gnbsim.git
     $ cd gnbsim
     $ make docker-build #this will create docker image

If you want to run gNBSim along with other SD-Core Network Functions then use [Aether onRamp](https://docs.aetherproject.org/master/onramp/overview.html) to deploy all network functions.

**Reference**: See how [Aether In a Box](https://gerrit.opencord.org/plugins/gitiles/aether-in-a-box/+/refs/heads/master) installs helm charts

## Option5: gNB simulator running standalone with single interface

![gNBSim Single Interface](/docs/images/Separate-cluster_Single_interface.jpg)

- Install gNB Simulator on any K8s cluster
- Multus cni needs to be enabled for the K8s cluster where bess-upf runs
- Make sure gNB Simulator can communicate with AMF & UPF
- *TODO* - New Makefile target will deploy just 5G control plane
- *TODO* - New Makefile target will deploy only gNB Simulator
- Single interface is used for user plane traffic towards UPF & as well traffic towards AMF
- defaultAs is configured per profile. This address is used to send data traffic during test
- configure AMF address or FQDN appropriately

> [!note]
>  Multiple gNB's can not be simulated since only 1 gNB will be able to use 2152 port


Following is example of override values

     config:
       gnbsim:
         singleInterface: true
         yamlCfgFiles:
           gnb.conf:
             configuration:
                gnbs: # pool of gNodeBs
                  gnb1:
                    n3IpAddr: "POD_IP" # set if singleInterface is true



## Option6: gNBSim running standalone with 2 or more interfaces

![Separate Cluster](/docs/images/Separate-cluster_2_interface.jpg)

- Install gNB Simulator on any K8s cluster
- Multus cni needs to be enabled on cluster. Required for bess-upf & gNB
- Make sure gNB Simulator can communicate with AMF & UPF
- *TODO* - New Makefile target will deploy just 5G control plane
- *TODO* - New Makefile target will deploy only gNB Simulator
- One interface is used for user plane traffic towards UPF
- Second interface is used to send traffic towards control plane (i.e. AMF).
- UPF network & default gateway is provided in the override values.
- Route to UPF network is added when POD comes up
- defaultAs is configured per profile. This address is used to send data traffic during test
- configure AMF address or FQDN appropriately

> [!note]
> Multiple gNB's in one simulator instance need more changes in helm chart. This is pending work.


To add UPF routes. Following is example of override values ::

     config:
       gnbsim:
         gnb:
           ip: 192.168.251.5/24 #user plane IP at gnb if 2 separate interface provided
         singleInterface: false
         networkTopo:
           - upfAddr: "192.168.252.3/32"
             upfGw: "192.168.251.1"


## Option7: Running gNBSim Standalone Application in or out of a Docker

![Standalone gNBSim](/docs/images/Standalone_gnbsim_1_interface.jpg)

Note that ``DATA-IFACE`` is ens1f0, this interface to be used for both control and data traffic

We need two VMs, in this example we call one is SD-Core VM, other one is Simulator VM
  * SD-Core VM: to Deploy AIAB
  * Simulator VM: to Run gnbsim process in or out of Docker

SD-Core VM Preparation:

- To Expose External IP and Port of amf service, update sd-core-5g-values.yaml ::

     amf:
       # use externalIP if you need to access your AMF from remote setup and you don't
       # want setup NodePort Service Type
       ngapp:
         externalIp: <DATA_IFACE_IP>
         nodePort: 38412
- Deploy 5g core with options DATA_IFACE=ens1f0 and ENABLE_GNBSIM=false, sample command::

     $ ENABLE_GNBSIM=false DATA_IFACE=ens1f0 CHARTS=release-2.0 make 5g-core
- Make sure that ``DATA_IFACE`` connected  with Simulator VM

Simulator VM Preparation

- Single interface is used for user plane traffic towards UPF
- Single interface is used to send traffic towards control plane (i.e. AMF).
- Checkout gnbsim code using the following command::

     $ git clone https://github.com/omec-project/gnbsim.git
- Install 'go' if you want to run with local executable ::

     $ wget  https://go.dev/dl/go1.19.linux-amd64.tar.gz
     $ sudo tar -xvf go1.19.linux-amd64.tar.gz
     $ mv go /usr/local
     $ export PATH=$PATH:/usr/local/go/bin
- To Compile the code locally, you can use below commands::

     $ ``go build`` or ``make docker-build``
- Add following route in routing table for sending traffic over ``DATA_IFACE`` interface ::

     $ ip route add 192.168.252.3 via <DATA-IFACE-IP-IN-SD-CORE-VM>
- Just to Make sure the data connectivity, ping UPF IP from ``DATA_IFACE``::

     $ ping 192.168.252.3 -I <DATA_IFACE>
- configure correct n2 and n3 addresses in config/gnbsim.yaml ::

        configuration:
             singleInterface: false #default value
             execInParallel: false #run all profiles in parallel
             gnbs: # pool of gNodeBs
                gnb1:
                   n2IpAddr: <DATA-IFACE-IP>># gNB N2 interface IP address used to connect to AMF
                   n2Port: 9487 # gNB N2 Port used to connect to AMF
                   n3IpAddr: <DATA-IFACE-IP> # gNB N3 interface IP address used to connect to UPF. when singleInterface mode is false
                   n3Port: 2152 # gNB N3 Port used to connect to UPF
                   name: gnb1 # gNB name that uniquely identify a gNB within application

- configure AMF address or FQDN appropriately in gnbsim.yaml ::

        configuration:
             singleInterface: false #default value
             execInParallel: false #run all profiles in parallel
             gnbs: # pool of gNodeBs
               gnb1:
                defaultAmf:
                   hostName:  # Host name of AMF
                   ipAddr: <AMF-SERVICE-EXTERNAL-IP> ># AMF Service external IP address in SD-Core VM
                   port: 38412 # AMF port
- Run gnbsim application using the following command::

   $ ./gnbsim -cfg config/gnbsim.yaml
        (or)
- Install Docker and run gnbsim inside a Docker with Docker hub Image or locally created Image ::

   $ docker run --privileged -it -v ~/gnbsim/config:/gnbsim/config --net=host <Docker-Image> bash
   $ ./gnbsim -cfg config/gnbsim.yaml

Note: gnbsim docker images found at https://hub.docker.com/r/omecproject/5gc-gnbsim/tags
