package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct VolumeTimedReportList */
/* VolumeTimedReportItem */
type VolumeTimedReportList struct {
	List []VolumeTimedReportItem `aper:"valueExt,sizeLB:1,sizeUB:2"`
}
