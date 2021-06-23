package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

type PDUSessionResourceSetupUnsuccessfulTransfer struct {
	Cause                  Cause                                                                        `aper:"valueLB:0,valueUB:5"`
	CriticalityDiagnostics *CriticalityDiagnostics                                                      `aper:"valueExt,optional"`
	IEExtensions           *ProtocolExtensionContainerPDUSessionResourceSetupUnsuccessfulTransferExtIEs `aper:"optional"`
}
