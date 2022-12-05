package utils

import "github.com/prometheus/client_golang/prometheus"

var (
	selflog                       *SLOG
	FunctionDurationTimeCostTotal *prometheus.GaugeVec
	FunctionDurationTimeCostCount *prometheus.GaugeVec
)

const (
	retryMark                   = "__ERROR_TO_RETRY__"
	CtxKey_RequestID CtxKeyType = "request_id"
	CtxKey_Logger    CtxKeyType = "logger"
)
