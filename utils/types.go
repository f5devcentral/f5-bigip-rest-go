package utils

import "log"

type SLOG struct {
	level     int
	requestID string
	loggers   map[int]*log.Logger
}

type CtxKeyType string
