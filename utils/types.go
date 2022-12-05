package utils

import "log"

type SLOG struct {
	requestID   string
	infoLogger  *log.Logger
	debugLogger *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	traceLogger *log.Logger
}

type CtxKeyType string
