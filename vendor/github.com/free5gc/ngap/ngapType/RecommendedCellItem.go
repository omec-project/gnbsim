package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type RecommendedCellItem struct {
	NGRANCGI         NGRANCGI                                             `aper:"valueLB:0,valueUB:2"`
	TimeStayedInCell *int64                                               `aper:"valueLB:0,valueUB:4095,optional"`
	IEExtensions     *ProtocolExtensionContainerRecommendedCellItemExtIEs `aper:"optional"`
}
