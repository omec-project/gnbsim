package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type TAIListForInactiveItem struct {
	TAI          TAI                                                     `aper:"valueExt"`
	IEExtensions *ProtocolExtensionContainerTAIListForInactiveItemExtIEs `aper:"optional"`
}
