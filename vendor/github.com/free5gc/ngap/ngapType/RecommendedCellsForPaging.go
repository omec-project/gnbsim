package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type RecommendedCellsForPaging struct {
	RecommendedCellList RecommendedCellList
	IEExtensions        *ProtocolExtensionContainerRecommendedCellsForPagingExtIEs `aper:"optional"`
}
