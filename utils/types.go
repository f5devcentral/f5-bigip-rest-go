package utils

import "log"

type SLOG struct {
	Level     int
	requestID string
	loggers   map[int]*log.Logger
}

type CtxKeyType string
