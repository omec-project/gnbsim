package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type QosFlowInformationItem struct {
	QosFlowIdentifier QosFlowIdentifier
	DLForwarding      *DLForwarding                                           `aper:"optional"`
	IEExtensions      *ProtocolExtensionContainerQosFlowInformationItemExtIEs `aper:"optional"`
}
