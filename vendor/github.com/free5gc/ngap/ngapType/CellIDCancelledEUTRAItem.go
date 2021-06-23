package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type CellIDCancelledEUTRAItem struct {
	EUTRACGI           EUTRACGI `aper:"valueExt"`
	NumberOfBroadcasts NumberOfBroadcasts
	IEExtensions       *ProtocolExtensionContainerCellIDCancelledEUTRAItemExtIEs `aper:"optional"`
}
