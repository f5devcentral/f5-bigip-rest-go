package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func NewLog() *SLOG {
	slog := SLOG{
		requestID: "",
		level:     LogLevel_INFO,
		loggers:   map[int]*log.Logger{},
	}
	for n, l := range levels {
		prefix := markLogPrefix(n, slog.requestID)
		slog.loggers[l] = log.New(os.Stdout, prefix, flags)
	}
	return &slog
}

func (slog *SLOG) WithRequestID(reqid string) *SLOG {
	slog.requestID = reqid
	for n, logger := range slog.loggers {
		prefix := markLogPrefix(itoaLevel(n), slog.requestID)
		logger.SetPrefix(prefix)
	}
	return slog
}

func (slog *SLOG) WithLevel(level string) *SLOG {
	slog.level = atoiLevel(level)
	return slog
}

func (slog *SLOG) Infof(format string, v ...interface{}) {
	if slog.level >= LogLevel_INFO {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.loggers[LogLevel_INFO].Printf(m)
		}
	}
}

func (slog *SLOG) Debugf(format string, v ...interface{}) {
	if slog.level >= LogLevel_DEBUG {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.loggers[LogLevel_DEBUG].Printf(m)
		}
	}
}

func (slog *SLOG) Warnf(format string, v ...interface{}) {
	if slog.level >= LogLevel_WARN {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.loggers[LogLevel_WARN].Printf(m)
		}
	}
}

func (slog *SLOG) Errorf(format string, v ...interface{}) {
	if slog.level >= LogLevel_ERROR {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.loggers[LogLevel_ERROR].Printf(m)
		}
	}
}

func (slog *SLOG) Tracef(format string, v ...interface{}) {
	if slog.level >= LogLevel_TRACE {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.loggers[LogLevel_TRACE].Printf(m)
		}
	}
}

func atoiLevel(level string) int {
	if l, ok := levels[level]; ok {
		return l
	} else {
		return LogLevel_INFO
	}
}

func itoaLevel(level int) string {
	for k, v := range levels {
		if level == v {
			return k
		}
	}
	return LogLevel_Type_INFO
}

func markLogPrefix(level, reqid string) string {
	lp := fmt.Sprintf("%7s", "["+strings.ToUpper(level)+"]")
	rp := fmt.Sprintf("[%s]", reqid)
	return fmt.Sprintf("%s %s ", lp, rp)
}
