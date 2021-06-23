package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type QosFlowAcceptedItem struct {
	QosFlowIdentifier QosFlowIdentifier
	IEExtensions      *ProtocolExtensionContainerQosFlowAcceptedItemExtIEs `aper:"optional"`
}
