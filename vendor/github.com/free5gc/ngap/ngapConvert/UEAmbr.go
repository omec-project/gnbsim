package ngapConvert

import (
	"strconv"
	"strings"

	"github.com/free5gc/ngap/logger"
)

func UEAmbrToInt64(modelAmbr string) int64 {
	tok := strings.Split(modelAmbr, " ")
	if ambr, err := strconv.ParseFloat(tok[0], 64); err != nil {
		logger.NgapLog.Warnf("Parse AMBR failed %+v", err)
		return int64(0)
	} else {
		return int64(ambr * getUnit(tok[1]))
	}
}

func getUnit(unit string) float64 {
	switch unit {
	case "bps":
		return 1.0
	case "Kbps":
		return 1000.0
	case "Mbps":
		return 1000000.0
	case "Gbps":
		return 1000000000.0
	case "Tbps":
		return 1000000000000.0
	}
	return 1.0
}
