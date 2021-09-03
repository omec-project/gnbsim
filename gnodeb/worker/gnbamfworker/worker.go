// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package gnbamfworker

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/gnodeb/context"

	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
)

// dispatch decodes an incoming NGAP message and routes it to the corresponding
// handlers or a GnbCpUe
func HandleMessage(gnb *context.GNodeB, amf *context.GnbAmf, pkt []byte) error {
	// decoding the incoming packet
	pdu, err := ngap.Decoder(pkt)
	if err != nil {
		return fmt.Errorf("NGAP decode error : %+v", err)
	}

	// routing to correct handlers
	switch pdu.Present {
	case ngapType.NGAPPDUPresentInitiatingMessage:
		initiatingMessage := pdu.InitiatingMessage
		if initiatingMessage == nil {
			return fmt.Errorf("UnSuccessful Outcome is nil")
		}
		switch initiatingMessage.ProcedureCode.Value {
		case ngapType.ProcedureCodeDownlinkNASTransport:
			HandleDownlinkNasTransport(gnb, amf, pdu)
		case ngapType.ProcedureCodeInitialContextSetup:
			HandleInitialContextSetupRequest(gnb, amf, pdu)
		}
	case ngapType.NGAPPDUPresentSuccessfulOutcome:
		successfulOutcome := pdu.SuccessfulOutcome
		if successfulOutcome == nil {
			return fmt.Errorf("successful Outcome is nil")
		}
		switch successfulOutcome.ProcedureCode.Value {
		case ngapType.ProcedureCodeNGSetup:
			HandleNgSetupResponse(amf, pdu)
		}
	case ngapType.NGAPPDUPresentUnsuccessfulOutcome:
		unsuccessfulOutcome := pdu.UnsuccessfulOutcome
		if unsuccessfulOutcome == nil {
			return fmt.Errorf("UnSuccessful Outcome is nil")
		}
		switch unsuccessfulOutcome.ProcedureCode.Value {
		case ngapType.ProcedureCodeNGSetup:
			HandleNgSetupFailure(amf, pdu)
		}
	}

	return nil
}

func SendToGnbUe(gnbue *context.GnbUe, event common.EventType, ngapPdu *ngapType.NGAPPDU) {
	amfmsg := common.N2Message{}
	amfmsg.Event = event
	amfmsg.Interface = common.N2_INTERFACE
	amfmsg.NgapPdu = ngapPdu
	gnbue.ReadChan <- &amfmsg
}
