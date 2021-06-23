package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type QosFlowAddOrModifyRequestItem struct {
	QosFlowIdentifier         QosFlowIdentifier
	QosFlowLevelQosParameters *QosFlowLevelQosParameters                                     `aper:"valueExt,optional"`
	ERABID                    *ERABID                                                        `aper:"optional"`
	IEExtensions              *ProtocolExtensionContainerQosFlowAddOrModifyRequestItemExtIEs `aper:"optional"`
}
