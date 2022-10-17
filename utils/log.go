package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type SLOG struct {
	requestID   string
	infoLogger  *log.Logger
	debugLogger *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	traceLogger *log.Logger
}

func SetupLog(reqid, level string) SLOG {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lmsgprefix
	levels := []string{"error", "warn", "info", "debug", "trace"}
	if !Contains(levels, level) {
		panic("invalid logging level: " + level)
	}
	slog := SLOG{
		requestID: reqid,
	}
	for _, l := range levels {
		tobreak := false
		if l == level {
			tobreak = true
		}
		fmtReqId := ""
		if reqid != "" {
			fmtReqId = fmt.Sprintf("[%s] ", reqid)
		}
		switch l {
		case "trace":
			slog.traceLogger = log.New(os.Stdout, "[TRACE] "+fmtReqId, flags)
		case "debug":
			slog.debugLogger = log.New(os.Stdout, "[DEBUG] "+fmtReqId, flags)
		case "info":
			slog.infoLogger = log.New(os.Stdout, "[INFO]  "+fmtReqId, flags)
		case "warn":
			slog.warnLogger = log.New(os.Stdout, "[WARN]  "+fmtReqId, flags)
		case "error":
			slog.errorLogger = log.New(os.Stderr, "[ERROR] "+fmtReqId, flags)
		}
		if tobreak {
			break
		}
	}
	return slog
}

func (slog *SLOG) Infof(format string, v ...interface{}) {
	if slog.infoLogger != nil {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.infoLogger.Printf(m)
		}
	}
}

func (slog *SLOG) Debugf(format string, v ...interface{}) {
	if slog.debugLogger != nil {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.debugLogger.Printf(m)
		}
	}
}

func (slog *SLOG) Warnf(format string, v ...interface{}) {
	if slog.warnLogger != nil {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.warnLogger.Printf(m)
		}
	}
}

func (slog *SLOG) Errorf(format string, v ...interface{}) {
	if slog.errorLogger != nil {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.errorLogger.Printf(m)
		}
	}
}

func (slog *SLOG) Tracef(format string, v ...interface{}) {
	if slog.traceLogger != nil {
		msg := fmt.Sprintf(format, v...)
		for _, m := range strings.Split(msg, "\n") {
			slog.traceLogger.Printf(m)
		}
	}
}
