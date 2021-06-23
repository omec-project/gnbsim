package ngapType

// Need to import "github.com/free5gc/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct CompletedCellsInTAI_EUTRA */
/* CompletedCellsInTAIEUTRAItem */
type CompletedCellsInTAIEUTRA struct {
	List []CompletedCellsInTAIEUTRAItem `aper:"valueExt,sizeLB:1,sizeUB:65535"`
}
