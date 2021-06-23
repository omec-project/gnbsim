package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	TraceDepthPresentMinimum                               aper.Enumerated = 0
	TraceDepthPresentMedium                                aper.Enumerated = 1
	TraceDepthPresentMaximum                               aper.Enumerated = 2
	TraceDepthPresentMinimumWithoutVendorSpecificExtension aper.Enumerated = 3
	TraceDepthPresentMediumWithoutVendorSpecificExtension  aper.Enumerated = 4
	TraceDepthPresentMaximumWithoutVendorSpecificExtension aper.Enumerated = 5
)

type TraceDepth struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:5"`
}
