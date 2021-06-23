package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	CancelAllWarningMessagesPresentTrue aper.Enumerated = 0
)

type CancelAllWarningMessages struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:0"`
}
