package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type MobilityRestrictionList struct {
	ServingPLMN              PLMNIdentity
	EquivalentPLMNs          *EquivalentPLMNs                                         `aper:"optional"`
	RATRestrictions          *RATRestrictions                                         `aper:"optional"`
	ForbiddenAreaInformation *ForbiddenAreaInformation                                `aper:"optional"`
	ServiceAreaInformation   *ServiceAreaInformation                                  `aper:"optional"`
	IEExtensions             *ProtocolExtensionContainerMobilityRestrictionListExtIEs `aper:"optional"`
}
