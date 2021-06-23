package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	RedirectionVoiceFallbackPresentPossible    aper.Enumerated = 0
	RedirectionVoiceFallbackPresentNotPossible aper.Enumerated = 1
)

type RedirectionVoiceFallback struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:1"`
}
