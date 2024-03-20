// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnbamfworker

import (
	"fmt"

	"github.com/omec-project/gnbsim/common"
	gnbctx "github.com/omec-project/gnbsim/gnodeb/context"
	"github.com/omec-project/ngap"
	"github.com/omec-project/ngap/ngapType"
)

/* HandleMessage decodes an incoming NGAP message and routes it to the
 * corresponding handlers
 */
func HandleMessage(gnb *gnbctx.GNodeB, amf *gnbctx.GnbAmf, pkt []byte, id uint64) error {
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
			return fmt.Errorf("Initiatiting Message is nil")
		}
		switch initiatingMessage.ProcedureCode.Value {
		case ngapType.ProcedureCodeDownlinkNASTransport:
			HandleDownlinkNasTransport(gnb, amf, pdu, id)
		case ngapType.ProcedureCodeInitialContextSetup:
			HandleInitialContextSetupRequest(gnb, amf, pdu, id)
		case ngapType.ProcedureCodePDUSessionResourceSetup:
			HandlePduSessResourceSetupRequest(gnb, amf, pdu, id)
		case ngapType.ProcedureCodePDUSessionResourceRelease:
			HandlePduSessResourceReleaseCommand(gnb, amf, pdu, id)
		case ngapType.ProcedureCodeUEContextRelease:
			HandleUeCtxReleaseCommand(gnb, amf, pdu, id)
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

func SendToGnbUe(gnbue *gnbctx.GnbCpUe, event common.EventType, ngapPdu *ngapType.NGAPPDU, id uint64) {
	amfmsg := common.N2Message{}
	amfmsg.Event = event
	amfmsg.NgapPdu = ngapPdu
	amfmsg.Id = id
	gnbue.ReadChan <- &amfmsg
}
