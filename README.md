# f5-bigip-rest

## Repository Introduction

The f5-bigip-rest repository encapsulates BIG-IP iControlRest calls in a simple and usable way. It contains two separate modules: `bigip` and `utils`

* `bigip` module can execute various BIG-IP iControlRest commands in the form of transactions, and the list of currently supported resources can be found [here](./bigip/utils.go).
* `utils` module encapsulates some necessary common objects and functions, such as logging, Prometheus monitoring, and HTTPRequest capabilities. See below for their usages.

## Module Usages

### `bigip` module:

*More details about the following code, see [here](./examples/bigip/bigip_deploy.go).*

```golang
package main

import (
	// import bigip module
	f5_bigip "gitee.com/zongzw/f5-bigip-rest/bigip"
)

func main() {
	// instanlize bigip for icontrol execution
	bc := f5_bigip.BIGIPContext{
		BIGIP:   *(f5_bigip.New("https://1.2.3.4", "admin", "password")),
		Context: context.TODO(),
	}

	var ncfgs map[string]interface{}
	partition := "mynamespace"
	// See the whole configs from ./examples/bigip/bigip_deploy.go
	configs := fmt.Sprintf(`
		{
			"service_name_app": {
				"ltm/monitor/http/service_name_monitor": {
					"adaptive": "disabled",
					"adaptiveDivergenceMilliseconds": 500,
					...
					"upInterval": 0
				},
				"ltm/pool/service_name_pool": {
					"allowNat": "yes",
					"allowSnat": "yes",
					...
					"slowRampTime": 10
				},
				"ltm/profile/http/service_name_httpprofile": {
					"acceptXff": "disabled",
					...
					"webSocketsEnabled": false
				},
				"ltm/profile/one-connect/service_name_oneconnectprofile": {
					"idleTimeoutOverride": 0,
					"limitType": "none",
					..
				}
			}
		}
	`, partition, partition, partition, partition)

	// convert the resource string to map[string]interface{}
	if err := json.Unmarshal([]byte(configs), &ncfgs); err != nil {
		fmt.Printf("Failed to deploy resources: %s\n", err.Error())
		os.Exit(1)
	}

	// Create the partition
	if err := bc.DeployPartition(partition); err != nil {
		fmt.Printf("failed to create partition: %s: %s\n", partition, err.Error())
		panic(err)
	}

	// generate the RestRequest list
	if cmds, err := bc.GenRestRequests(partition, nil, &ncfgs); err != nil {
		fmt.Printf("failed to generate rest requests for deploying: %s\n", err.Error())
		panic(err)
	} else {
		// execute the RestRequest list in a transaction
		if err := bc.DoRestRequests(cmds); err != nil {
			fmt.Printf("failed to deploy with rest requests: %s\n", err.Error())
			panic(err)
		} else {
			fmt.Println("deployed requests.")
			kinds := []string{"ltm/virtual", "ltm/pool", "ltm/monitor/http"}
			// get and verify the partition and resources are created as expected.
			if existings, err := bc.GetExistingResources(partition, kinds); err != nil {
				fmt.Printf("failed to get existing resources of %s: %s\n", kinds, err.Error())
				panic(err)
			} else {
				b, _ := json.MarshalIndent(existings, "", "  ")
				fmt.Println(string(b))
			}
		}
	}
}
```

in the above code, we demonstrate [`BIGIPContext`](./bigip/types.go) important functions' usage.

### `utils` module

*More details about the following code, see [here](./examples/utils/utils_sample.go).*
```golang
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
	uid := uuid.New().String()

	// get a new logger instance.
	slog := utils.NewLog()
	// set log level and request id for tracing
	slog.WithLevel(utils.LogLevel_Type_DEBUG).WithRequestID(uid)
	
	slog.Debugf("hello there.")
	
	// embed logger into a context for later using.
	ctx := context.WithValue(context.TODO(), utils.CtxKey_Logger, slog)

	// register prometheus metric names.
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
	// collect prometheus metrics
	defer utils.TimeItToPrometheus()()

	// get logger from context.
	slog := utils.LogFromContext(ctx)

	<-time.After(100 * time.Millisecond)
	slog.Debugf("called function promTest")
}

```

Further, it's easy to find some detailed usages from [f5-tool-deploy-rest](https://gitee.com/zongzw/f5-tool-deploy-rest).