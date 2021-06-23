package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct PDUSessionResourceFailedToSetupListPSReq */
/* PDUSessionResourceFailedToSetupItemPSReq */
type PDUSessionResourceFailedToSetupListPSReq struct {
	List []PDUSessionResourceFailedToSetupItemPSReq `aper:"valueExt,sizeLB:1,sizeUB:256"`
}
