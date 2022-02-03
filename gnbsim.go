// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"fmt"
	"gnbsim/deregister"
	"gnbsim/duplicateregistration"
	"gnbsim/gutiregistration"
	"gnbsim/loadsub"
	"gnbsim/n2handover"
	"gnbsim/paging"
	"gnbsim/pdusessionrelease"
	"gnbsim/register"
	"gnbsim/resynchronisation"
	"gnbsim/servicereq"
	"gnbsim/xnhandover"
	"net"
	"os"

	"github.com/free5gc/MongoDBLibrary"
)

func main() {
	fmt.Println("Main function")
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "(loadsubs | register|deregister|xnhandover|paging|n2handover|servicereq|servicereqmacfail|resynchronisation|gutiregistration|duplicateregistration|pdusessionrelease)")
		return
	}
	testcase := os.Args[1]

	fmt.Println("argsWithoutProg ", testcase)

	ranIpAddr := os.Getenv("POD_IP")
	fmt.Println("Hello World from RAN - ", ranIpAddr)

	// RAN connect to AMF
	addrs, err := net.LookupHost("amf")
	if err != nil {
		fmt.Println("Failed to resolve amf")
		return
	}
	amfIpAddr := addrs[0]
	fmt.Println("AMF address - ", amfIpAddr)

	addrs, err = net.LookupHost("upf")
	if err != nil {
		fmt.Println("Failed to resolve upf")
		return
	}
	upfIpAddr := addrs[0]
	fmt.Println("UPF address - ", upfIpAddr)

	upfIpAddr = "192.168.252.3"
	fmt.Println("UPF address - ", upfIpAddr)

	addrs, err = net.LookupHost("mongodb")
	if err != nil {
		fmt.Println("Failed to resolve mongodb")
		return
	}
	mongodbIpAddr := addrs[0]
	fmt.Println("mongodb address - ", mongodbIpAddr)

	dbName := "free5gc"
	dbUrl := "mongodb://mongodb:27017"
	MongoDBLibrary.SetMongoDB(dbName, dbUrl)
	fmt.Println("Connected to MongoDB ")
	ranUIpAddr := "192.168.251.5"

	switch testcase {
	case "register":
		{
			fmt.Println("test register")
			register.Register_test(ranUIpAddr, ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "deregister":
		{
			fmt.Println("test deregister")
			deregister.Deregister_test(ranIpAddr, amfIpAddr)
		}
	case "pdusessionrelease":
		{
			fmt.Println("test pdusessionrelease")
			pdusessionrelease.PduSessionRelease_test(ranIpAddr, amfIpAddr)
		}
	case "duplicateregistration":
		{
			fmt.Println("test duplicateregistration")
			duplicateregistration.DuplicateRegistration_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "gutiregistration":
		{
			fmt.Println("test gutiregistration")
			gutiregistration.Gutiregistration_test(ranIpAddr, amfIpAddr)
		}
	case "n2handover":
		{
			fmt.Println("test n2handover")
			n2handover.N2Handover_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "paging":
		{
			fmt.Println("test paging")
			paging.Paging_test(ranIpAddr, amfIpAddr)
		}
	case "resynchronisation":
		{
			fmt.Println("test resynchronisation")
			resynchronisation.Resychronisation_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "servicereqmacfail":
		{
			fmt.Println("test servicereq macfail")
			servicereq.Servicereq_macfail_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "servicereq":
		{
			fmt.Println("test servicereq")
			servicereq.Servicereq_test(ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "xnhandover":
		{
			fmt.Println("test xnhandover")
			xnhandover.Xnhandover_test(ranUIpAddr, ranIpAddr, upfIpAddr, amfIpAddr)
		}
	case "loadsubs":
		{
			fmt.Println("loading subscribers in DB")
			loadsub.LoadSubscriberData(10)
		}
	}

	return
}
