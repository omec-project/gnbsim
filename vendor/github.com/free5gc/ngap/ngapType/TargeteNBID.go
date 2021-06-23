package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type TargeteNBID struct {
	GlobalENBID    GlobalNgENBID                                `aper:"valueExt"`
	SelectedEPSTAI EPSTAI                                       `aper:"valueExt"`
	IEExtensions   *ProtocolExtensionContainerTargeteNBIDExtIEs `aper:"optional"`
}
