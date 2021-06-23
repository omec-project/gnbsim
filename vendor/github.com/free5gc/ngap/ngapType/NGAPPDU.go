package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	NGAPPDUPresentNothing int = iota /* No components present */
	NGAPPDUPresentInitiatingMessage
	NGAPPDUPresentSuccessfulOutcome
	NGAPPDUPresentUnsuccessfulOutcome
	/* Extensions may appear below */
)

type NGAPPDU struct {
	Present             int
	InitiatingMessage   *InitiatingMessage
	SuccessfulOutcome   *SuccessfulOutcome
	UnsuccessfulOutcome *UnsuccessfulOutcome
}
