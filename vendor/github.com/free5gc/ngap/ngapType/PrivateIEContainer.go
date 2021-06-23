package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct PrivateIE_Container_6722P0 */
/* PrivateMessageIEs */
type PrivateIEContainerPrivateMessageIEs struct {
	List []PrivateMessageIEs `aper:"sizeLB:1,sizeUB:65535"`
}
