// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"bytes"
	"fmt"

	realuectx "github.com/omec-project/gnbsim/realue/context"
	"github.com/omec-project/gnbsim/util/nastestpacket"
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
		return nil, fmt.Errorf("encode failed:+%v", err)
	}

	return data.Bytes(), nil
}
