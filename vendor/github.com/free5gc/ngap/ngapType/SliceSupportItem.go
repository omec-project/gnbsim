package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type SliceSupportItem struct {
	SNSSAI       SNSSAI                                            `aper:"valueExt"`
	IEExtensions *ProtocolExtensionContainerSliceSupportItemExtIEs `aper:"optional"`
}
