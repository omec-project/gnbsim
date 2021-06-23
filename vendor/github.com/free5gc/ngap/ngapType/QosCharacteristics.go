package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	QosCharacteristicsPresentNothing int = iota /* No components present */
	QosCharacteristicsPresentNonDynamic5QI
	QosCharacteristicsPresentDynamic5QI
	QosCharacteristicsPresentChoiceExtensions
)

type QosCharacteristics struct {
	Present          int
	NonDynamic5QI    *NonDynamic5QIDescriptor `aper:"valueExt"`
	Dynamic5QI       *Dynamic5QIDescriptor    `aper:"valueExt"`
	ChoiceExtensions *ProtocolIESingleContainerQosCharacteristicsExtIEs
}
