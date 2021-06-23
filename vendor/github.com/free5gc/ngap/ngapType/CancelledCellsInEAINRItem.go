package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type CancelledCellsInEAINRItem struct {
	NRCGI              NRCGI `aper:"valueExt"`
	NumberOfBroadcasts NumberOfBroadcasts
	IEExtensions       *ProtocolExtensionContainerCancelledCellsInEAINRItemExtIEs `aper:"optional"`
}
