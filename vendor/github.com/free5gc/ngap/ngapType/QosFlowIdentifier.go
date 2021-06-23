package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type QosFlowIdentifier struct {
	Value int64 `aper:"valueExt,valueLB:0,valueUB:63"`
}
