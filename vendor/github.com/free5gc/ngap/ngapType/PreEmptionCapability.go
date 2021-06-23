package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	PreEmptionCapabilityPresentShallNotTriggerPreEmption aper.Enumerated = 0
	PreEmptionCapabilityPresentMayTriggerPreEmption      aper.Enumerated = 1
)

type PreEmptionCapability struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:1"`
}
