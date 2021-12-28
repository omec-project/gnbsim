// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package nas

import (
	"bytes"
	"fmt"
	"gnbsim/realue/context"
	"gnbsim/util/nastestpacket"

	"github.com/free5gc/nas/nasConvert"
	"github.com/free5gc/nas/nasMessage"
)

func GetServiceRequest(ue *context.RealUe) ([]byte, error) {

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
		return nil, fmt.Errorf("encode failed:", err)
	}

	return data.Bytes(), nil
}
