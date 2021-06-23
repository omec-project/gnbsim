package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	OverloadActionPresentRejectNonEmergencyMoDt                                    aper.Enumerated = 0
	OverloadActionPresentRejectRrcCrSignalling                                     aper.Enumerated = 1
	OverloadActionPresentPermitEmergencySessionsAndMobileTerminatedServicesOnly    aper.Enumerated = 2
	OverloadActionPresentPermitHighPrioritySessionsAndMobileTerminatedServicesOnly aper.Enumerated = 3
)

type OverloadAction struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:3"`
}
