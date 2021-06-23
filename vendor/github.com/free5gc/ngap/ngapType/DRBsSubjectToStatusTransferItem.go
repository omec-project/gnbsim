package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type DRBsSubjectToStatusTransferItem struct {
	DRBID       DRBID
	DRBStatusUL DRBStatusUL                                                      `aper:"valueLB:0,valueUB:2"`
	DRBStatusDL DRBStatusDL                                                      `aper:"valueLB:0,valueUB:2"`
	IEExtension *ProtocolExtensionContainerDRBsSubjectToStatusTransferItemExtIEs `aper:"optional"`
}
