package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type SecurityContext struct {
	NextHopChainingCount NextHopChainingCount
	NextHopNH            SecurityKey
	IEExtensions         *ProtocolExtensionContainerSecurityContextExtIEs `aper:"optional"`
}
