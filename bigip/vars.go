package f5_bigip

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// slog                       *utils.SLOG
	ResOrder                   []string
	BIGIPiControlTimeCostTotal *prometheus.GaugeVec
	BIGIPiControlTimeCostCount *prometheus.GaugeVec
)

const TmUriPrefix = "/mgmt/tm"
