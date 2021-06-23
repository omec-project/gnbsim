package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	TimeToWaitPresentV1s  aper.Enumerated = 0
	TimeToWaitPresentV2s  aper.Enumerated = 1
	TimeToWaitPresentV5s  aper.Enumerated = 2
	TimeToWaitPresentV10s aper.Enumerated = 3
	TimeToWaitPresentV20s aper.Enumerated = 4
	TimeToWaitPresentV60s aper.Enumerated = 5
)

type TimeToWait struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:5"`
}
