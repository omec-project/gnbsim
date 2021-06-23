package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct UPTransportLayerInformationPairList */
/* UPTransportLayerInformationPairItem */
type UPTransportLayerInformationPairList struct {
	List []UPTransportLayerInformationPairItem `aper:"valueExt,sizeLB:1,sizeUB:3"`
}
