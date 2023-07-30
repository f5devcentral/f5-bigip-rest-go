package utils

import (
	"log"
	"sync"
)

type SLOG struct {
	Level     int
	requestID string
	loggers   map[int]*log.Logger
}

type CtxKeyType string

type DeployQueue struct {
	Items []interface{}
	found chan bool
	mutex sync.Mutex
}
