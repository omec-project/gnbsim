package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type CompletedCellsInTAIEUTRAItem struct {
	EUTRACGI     EUTRACGI                                                      `aper:"valueExt"`
	IEExtensions *ProtocolExtensionContainerCompletedCellsInTAIEUTRAItemExtIEs `aper:"optional"`
}
