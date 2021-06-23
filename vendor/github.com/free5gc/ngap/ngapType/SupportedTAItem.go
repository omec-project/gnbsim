package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type SupportedTAItem struct {
	TAC               TAC
	BroadcastPLMNList BroadcastPLMNList
	IEExtensions      *ProtocolExtensionContainerSupportedTAItemExtIEs `aper:"optional"`
}
