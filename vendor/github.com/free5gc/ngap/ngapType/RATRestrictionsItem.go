package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type RATRestrictionsItem struct {
	PLMNIdentity              PLMNIdentity
	RATRestrictionInformation RATRestrictionInformation
	IEExtensions              *ProtocolExtensionContainerRATRestrictionsItemExtIEs `aper:"optional"`
}
