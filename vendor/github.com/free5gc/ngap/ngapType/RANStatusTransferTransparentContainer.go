package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type RANStatusTransferTransparentContainer struct {
	DRBsSubjectToStatusTransferList DRBsSubjectToStatusTransferList
	IEExtensions                    *ProtocolExtensionContainerRANStatusTransferTransparentContainerExtIEs `aper:"optional"`
}
