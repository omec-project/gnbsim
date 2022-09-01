// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"bytes"
	"fmt"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/gnbsim/realue/util"
	"github.com/omec-project/gnbsim/util/nastestpacket"
	"github.com/omec-project/nas/nasTestpacket"
	"github.com/omec-project/nas/nasType"

	"github.com/omec-project/nas/nasConvert"
	"github.com/omec-project/nas/nasMessage"
)

func GetServiceRequest(ue *realuectx.RealUe) ([]byte, error) {

	nasMsg := nastestpacket.BuildServiceRequest(nasMessage.ServiceTypeData)
	serviceRequest := nasMsg.GmmMessage.ServiceRequest

	guti := nasConvert.GutiToNas(ue.Guti)
	serviceRequest.SetTypeOfIdentity(nasMessage.MobileIdentity5GSType5gSTmsi)
	serviceRequest.SetAMFSetID(guti.GetAMFSetID())
	serviceRequest.SetAMFPointer(guti.GetAMFPointer())
	serviceRequest.SetTMSI5G(guti.GetTMSI5G())
	serviceRequest.SetNasKeySetIdentifiler(uint8(ue.NgKsi.Ksi))

	data := new(bytes.Buffer)
	err := nasMsg.GmmMessageEncode(data)
	if err != nil {
		return nil, fmt.Errorf("encode failed:%v", err)
	}

	return data.Bytes(), nil
}

func GetRegistrationRequest(ue *realuectx.RealUe) ([]byte, error) {
	ueSecurityCapability := ue.GetUESecurityCapability()

	var err error
	ue.Suci, err = util.SupiToSuci(ue.Supi, ue.Plmn)
	if err != nil {
		return nil, fmt.Errorf("failed to derive suci from supi")
	}
	mobileId5GS := nasType.MobileIdentity5GS{
		Len:    uint16(len(ue.Suci)), // suci
		Buffer: ue.Suci,
	}

	ue.Log.Traceln("Generating Registration Request Message")
	nasPdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileId5GS, nil, ueSecurityCapability, nil, nil, nil)

	return nasPdu, nil
}
