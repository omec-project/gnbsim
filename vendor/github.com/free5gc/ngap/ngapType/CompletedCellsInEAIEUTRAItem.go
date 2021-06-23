package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type CompletedCellsInEAIEUTRAItem struct {
	EUTRACGI     EUTRACGI                                                      `aper:"valueExt"`
	IEExtensions *ProtocolExtensionContainerCompletedCellsInEAIEUTRAItemExtIEs `aper:"optional"`
}
