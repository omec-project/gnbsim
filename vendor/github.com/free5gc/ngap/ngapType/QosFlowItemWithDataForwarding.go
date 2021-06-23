package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type QosFlowItemWithDataForwarding struct {
	QosFlowIdentifier      QosFlowIdentifier
	DataForwardingAccepted *DataForwardingAccepted                                        `aper:"optional"`
	IEExtensions           *ProtocolExtensionContainerQosFlowItemWithDataForwardingExtIEs `aper:"optional"`
}
