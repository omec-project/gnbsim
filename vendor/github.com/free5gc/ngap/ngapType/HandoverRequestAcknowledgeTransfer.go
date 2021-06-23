package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type HandoverRequestAcknowledgeTransfer struct {
	DLNGUUPTNLInformation         UPTransportLayerInformation  `aper:"valueLB:0,valueUB:1"`
	DLForwardingUPTNLInformation  *UPTransportLayerInformation `aper:"valueLB:0,valueUB:1,optional"`
	SecurityResult                *SecurityResult              `aper:"valueExt,optional"`
	QosFlowSetupResponseList      QosFlowListWithDataForwarding
	QosFlowFailedToSetupList      *QosFlowListWithCause                                               `aper:"optional"`
	DataForwardingResponseDRBList *DataForwardingResponseDRBList                                      `aper:"optional"`
	IEExtensions                  *ProtocolExtensionContainerHandoverRequestAcknowledgeTransferExtIEs `aper:"optional"`
}
