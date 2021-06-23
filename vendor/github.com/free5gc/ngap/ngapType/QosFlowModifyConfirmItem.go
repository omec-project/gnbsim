package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type QosFlowModifyConfirmItem struct {
	QosFlowIdentifier QosFlowIdentifier
	IEExtensions      *ProtocolExtensionContainerQosFlowModifyConfirmItemExtIEs `aper:"optional"`
}
