package main

import (
	"context"
	"net/http"
	"time"

	"gitee.com/zongzw/f5-bigip-rest/utils"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// get a new logger instance and set log level and request id for tracing.
	slog := utils.NewLog()
	uid := uuid.New().String()
	slog.WithLevel(utils.LogLevel_Type_DEBUG).WithRequestID(uid)
	slog.Debugf("hello there.")
	// embed logger into a context for later using.
	ctx := context.WithValue(context.TODO(), utils.CtxKey_Logger, slog)

	// prometheus usage
	prometheus.MustRegister(utils.FunctionDurationTimeCostCount)
	prometheus.MustRegister(utils.FunctionDurationTimeCostTotal)
	// f5_bigip "gitee.com/zongzw/f5-bigip-rest/bigip"
	// prometheus.MustRegister(f5_bigip.BIGIPiControlTimeCostCount)
	// prometheus.MustRegister(f5_bigip.BIGIPiControlTimeCostTotal)

	callMonitoredFunc(ctx)

	http.Handle("/metrics", promhttp.Handler())
	func() {
		slog.Infof("Starting 8080 for prometheus monitoring..")
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()

	// By accessing http://localhost:8080, we can get the following metrics.
	// 	# HELP function_duration_timecost_count time cost count of functions
	// 	# TYPE function_duration_timecost_count gauge
	// 	function_duration_timecost_count{name="main.callMonitoredFunc"} 1
	// 	# HELP function_duration_timecost_total time cost total(in milliseconds) of functions
	// 	# TYPE function_duration_timecost_total gauge
	// 	function_duration_timecost_total{name="main.callMonitoredFunc"} 100
}

// callMonitoredFunc a test function
func callMonitoredFunc(ctx context.Context) {
	defer utils.TimeItToPrometheus()()
	slog := utils.LogFromContext(ctx)
	<-time.After(100 * time.Millisecond)
	slog.Debugf("called function promTest")
}
