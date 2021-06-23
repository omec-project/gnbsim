package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type EUTRACGI struct {
	PLMNIdentity      PLMNIdentity
	EUTRACellIdentity EUTRACellIdentity
	IEExtensions      *ProtocolExtensionContainerEUTRACGIExtIEs `aper:"optional"`
}
