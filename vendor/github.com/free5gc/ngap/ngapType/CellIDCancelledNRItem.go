package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type CellIDCancelledNRItem struct {
	NRCGI              NRCGI `aper:"valueExt"`
	NumberOfBroadcasts NumberOfBroadcasts
	IEExtensions       *ProtocolExtensionContainerCellIDCancelledNRItemExtIEs `aper:"optional"`
}
