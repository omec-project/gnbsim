package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	CauseNasPresentNormalRelease         aper.Enumerated = 0
	CauseNasPresentAuthenticationFailure aper.Enumerated = 1
	CauseNasPresentDeregister            aper.Enumerated = 2
	CauseNasPresentUnspecified           aper.Enumerated = 3
)

type CauseNas struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:3"`
}
