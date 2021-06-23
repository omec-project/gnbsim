package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	UserLocationInformationPresentNothing int = iota /* No components present */
	UserLocationInformationPresentUserLocationInformationEUTRA
	UserLocationInformationPresentUserLocationInformationNR
	UserLocationInformationPresentUserLocationInformationN3IWF
	UserLocationInformationPresentChoiceExtensions
)

type UserLocationInformation struct {
	Present                      int
	UserLocationInformationEUTRA *UserLocationInformationEUTRA `aper:"valueExt"`
	UserLocationInformationNR    *UserLocationInformationNR    `aper:"valueExt"`
	UserLocationInformationN3IWF *UserLocationInformationN3IWF `aper:"valueExt"`
	ChoiceExtensions             *ProtocolIESingleContainerUserLocationInformationExtIEs
}
