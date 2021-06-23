package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type PDUSessionResourceReleaseResponseTransfer struct {
	IEExtensions *ProtocolExtensionContainerPDUSessionResourceReleaseResponseTransferExtIEs `aper:"optional"`
}
