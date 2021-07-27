// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package loadsub

import (
	"fmt"
    "gnbsim/util/test" // AJAY - Change required 
	"github.com/free5gc/CommonConsumerTestData/UDM/TestGenAuthData"
	"github.com/omec-project/nas/security"
	"strconv"
)

func LoadSubscriberData(num int) {
	var baseImsi int = 2089300007487
    var i int
	for i = 0; i < num; i++ {
		servingPlmnId := "20893"
		imsi := baseImsi + i
		supi := "imsi-" + strconv.Itoa(imsi)
		ue := test.NewRanUeContext(supi, 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2)
		ue.AmfUeNgapId = 1
		ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
			TestGenAuthData.MilenageTestSet19.OPC, "")

		fmt.Println("Insert Auth Subscription data to MongoDB")
		test.InsertAuthSubscriptionToMongoDB(ue.Supi, ue.AuthenticationSubs)
		getData := test.GetAuthSubscriptionFromMongoDB(ue.Supi)
		if getData == nil {
			return
		}

		{
			fmt.Println("Insert Access & Mobility Subscription data to MongoDB")
			amData := test.GetAccessAndMobilitySubscriptionData()
			test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
			getData := test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
			if getData == nil {
				return
			}
		}
		{
			fmt.Println("Insert SMF Selection Subscription data to MongoDB")
			smfSelData := test.GetSmfSelectionSubscriptionData()
			test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
			getData := test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
			if getData == nil {
				return
			}
		}
		{
			fmt.Println("Insert Session Management Subscription data to MongoDB")
			smSelData := test.GetSessionManagementSubscriptionData()
			test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
			getData := test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
			if getData == nil {
				return
			}
		}
		{
			fmt.Println("Insert Access mobility Policy data to MongoDB")
			amPolicyData := test.GetAmPolicyData()
			test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
			getData := test.GetAmPolicyDataFromMongoDB(ue.Supi)
			if getData == nil {
				return
			}
		}
		{
			fmt.Println("Insert Session Management Policy data to MongoDB")
			smPolicyData := test.GetSmPolicyData()
			test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
			getData := test.GetSmPolicyDataFromMongoDB(ue.Supi)
			if getData == nil {
				return
			}
		}

	}
}
