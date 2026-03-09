// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/omec-project/gnbsim/util/ngapTestpacket"
	"github.com/omec-project/ngap"
)

func GetUplinkNASTransport(amfUeNgapID, ranUeNgapID int64, nasPdu []byte) ([]byte, error) {
	message := ngapTestpacket.BuildUplinkNasTransport(amfUeNgapID, ranUeNgapID, nasPdu)
	return ngap.Encoder(message)
}

func GetInitialContextSetupResponse(amfUeNgapID int64, ranUeNgapID int64) ([]byte, error) {
	message := ngapTestpacket.BuildInitialContextSetupResponseForRegistrationTest(amfUeNgapID, ranUeNgapID)

	return ngap.Encoder(message)
}

func GetInitialContextSetupResponseForServiceRequest(
	pduSessions []*ngapTestpacket.PduSession, amfUeNgapID int64,
	ranUeNgapID int64, ipv4 string,
) ([]byte, error) {
	message := ngapTestpacket.BuildInitialContextSetupResponse(pduSessions, amfUeNgapID, ranUeNgapID, ipv4, nil)
	return ngap.Encoder(message)
}

func GetPDUSessionResourceSetupResponse(pduSessions []*ngapTestpacket.PduSession,
	amfUeNgapID int64, ranUeNgapID int64, ipv4 string,
) ([]byte, error) {
	message := ngapTestpacket.BuildPDUSessionResourceSetupResponseForRegistrationTest(pduSessions, amfUeNgapID, ranUeNgapID, ipv4)
	return ngap.Encoder(message)
}

func GetPDUSessionResourceReleaseResponse(amfUeNgapID int64, ranUeNgapID int64) ([]byte, error) {
	message := ngapTestpacket.BuildPDUSessionResourceReleaseResponseForReleaseTest(amfUeNgapID, ranUeNgapID)
	return ngap.Encoder(message)
}
