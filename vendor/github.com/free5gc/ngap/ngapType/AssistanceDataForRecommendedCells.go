package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type AssistanceDataForRecommendedCells struct {
	RecommendedCellsForPaging RecommendedCellsForPaging                                          `aper:"valueExt"`
	IEExtensions              *ProtocolExtensionContainerAssistanceDataForRecommendedCellsExtIEs `aper:"optional"`
}
