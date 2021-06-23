package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type AllowedNSSAIItem struct {
	SNSSAI       SNSSAI                                            `aper:"valueExt"`
	IEExtensions *ProtocolExtensionContainerAllowedNSSAIItemExtIEs `aper:"optional"`
}
