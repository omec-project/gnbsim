package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	CausePresentNothing int = iota /* No components present */
	CausePresentRadioNetwork
	CausePresentTransport
	CausePresentNas
	CausePresentProtocol
	CausePresentMisc
	CausePresentChoiceExtensions
)

type Cause struct {
	Present          int
	RadioNetwork     *CauseRadioNetwork
	Transport        *CauseTransport
	Nas              *CauseNas
	Protocol         *CauseProtocol
	Misc             *CauseMisc
	ChoiceExtensions *ProtocolIESingleContainerCauseExtIEs
}
