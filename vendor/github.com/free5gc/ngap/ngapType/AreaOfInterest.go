package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type AreaOfInterest struct {
	AreaOfInterestTAIList     *AreaOfInterestTAIList                          `aper:"optional"`
	AreaOfInterestCellList    *AreaOfInterestCellList                         `aper:"optional"`
	AreaOfInterestRANNodeList *AreaOfInterestRANNodeList                      `aper:"optional"`
	IEExtensions              *ProtocolExtensionContainerAreaOfInterestExtIEs `aper:"optional"`
}
