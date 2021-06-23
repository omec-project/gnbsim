package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct EmergencyAreaIDCancelledEUTRA */
/* EmergencyAreaIDCancelledEUTRAItem */
type EmergencyAreaIDCancelledEUTRA struct {
	List []EmergencyAreaIDCancelledEUTRAItem `aper:"valueExt,sizeLB:1,sizeUB:65535"`
}
