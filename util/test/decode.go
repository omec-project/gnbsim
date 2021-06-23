package test

import (
	"github.com/free5gc/nas"
	"github.com/free5gc/ngap/ngapType"
    "fmt"
)

func GetNasPdu(ue *RanUeContext, msg *ngapType.DownlinkNASTransport) (m *nas.Message) {
	for _, ie := range msg.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDNASPDU {
			pkg := []byte(ie.Value.NASPDU.Value)
			m, err := NASDecode(ue, nas.GetSecurityHeaderType(pkg), pkg)
			if err != nil {
				return nil
			}
			return m
		}
	}
	return nil
}
func GetNasPduSetupRequest(ue *RanUeContext, msg *ngapType.PDUSessionResourceSetupRequest) (m *nas.Message) {
        for _, ie := range msg.ProtocolIEs.List {
                if ie.Id.Value == ngapType.ProtocolIEIDPDUSessionResourceSetupListSUReq {
                        x := ie.Value.PDUSessionResourceSetupListSUReq
                        for  _, ie1:= range x.List {
                                if ie1.PDUSessionNASPDU != nil  {
                                        fmt.Println("Found NAS PDU inside ResourceSEtupList")
                                        pkg := []byte(ie1.PDUSessionNASPDU.Value)
                                        m, err := NASDecode(ue, nas.GetSecurityHeaderType(pkg), pkg)
                                        fmt.Println("UE address - ", m.GmmMessage.DLNASTransport.Ipaddr)
                                        if err != nil {
                                                return nil
                                        }
                                        return m
                                }
                        }
                }
        }
        return nil
}
