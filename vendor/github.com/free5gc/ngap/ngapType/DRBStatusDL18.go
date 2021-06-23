package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type DRBStatusDL18 struct {
	DLCOUNTValue COUNTValueForPDCPSN18                          `aper:"valueExt"`
	IEExtension  *ProtocolExtensionContainerDRBStatusDL18ExtIEs `aper:"optional"`
}
