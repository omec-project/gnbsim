package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type SecondaryRATDataUsageReportTransfer struct {
	SecondaryRATUsageInformation *SecondaryRATUsageInformation                                        `aper:"valueExt,optional"`
	IEExtensions                 *ProtocolExtensionContainerSecondaryRATDataUsageReportTransferExtIEs `aper:"optional"`
}
