package f5_bigip

import (
	"gitee.com/zongzw/f5-bigip-rest/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	slog                       utils.SLOG
	ResOrder                   []string
	BIGIPiControlTimeCostTotal *prometheus.GaugeVec
	BIGIPiControlTimeCostCount *prometheus.GaugeVec
)
