package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type LastVisitedCellItem struct {
	LastVisitedCellInformation LastVisitedCellInformation                           `aper:"valueLB:0,valueUB:4"`
	IEExtensions               *ProtocolExtensionContainerLastVisitedCellItemExtIEs `aper:"optional"`
}
