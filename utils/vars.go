package utils

import "github.com/prometheus/client_golang/prometheus"

var (
	selflog                       *SLOG
	flags                         int
	levels                        map[string]int
	FunctionDurationTimeCostTotal *prometheus.GaugeVec
	FunctionDurationTimeCostCount *prometheus.GaugeVec
)

const (
	retryMark                   = "__ERROR_TO_RETRY__"
	CtxKey_RequestID CtxKeyType = "request_id"
	CtxKey_Logger    CtxKeyType = "logger"
)

const (
	LogLevel_ERROR = 1 << iota
	LogLevel_WARN
	LogLevel_INFO
	LogLevel_DEBUG
	LogLevel_TRACE
	LogLevel_Type_TRACE = "trace"
	LogLevel_Type_DEBUG = "debug"
	LogLevel_Type_INFO  = "info"
	LogLevel_Type_WARN  = "warn"
	LogLevel_Type_ERROR = "error"
)
