package utils

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	FunctionDurationTimeCostTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "function_duration_timecost_total",
			Help: "time cost total(in milliseconds) of functions",
		},
		[]string{"name"},
	)
	FunctionDurationTimeCostCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "function_duration_timecost_count",
			Help: "time cost count of functions",
		},
		[]string{"name"},
	)

	flags = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lmsgprefix
	levels = map[string]int{
		LogLevel_Type_TRACE: LogLevel_TRACE,
		LogLevel_Type_DEBUG: LogLevel_DEBUG,
		LogLevel_Type_INFO:  LogLevel_INFO,
		LogLevel_Type_WARN:  LogLevel_WARN,
		LogLevel_Type_ERROR: LogLevel_ERROR,
	}
	selflog = NewLog()
}

func TimeIt(slog *SLOG) func(format string, a ...interface{}) int64 {
	return TimeItWithLogFunc(slog.Debugf, 3)
}

func TimeItTrace(slog *SLOG) func(format string, a ...interface{}) int64 {
	return TimeItWithLogFunc(slog.Tracef, 3)
}

func TimeItToPrometheus() func() {
	start := time.Now()

	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])

	return func() {
		tc := time.Since(start)
		FunctionDurationTimeCostTotal.WithLabelValues(f.Name()).Add(float64(tc.Milliseconds()))
		FunctionDurationTimeCostCount.WithLabelValues(f.Name()).Inc()
	}
}

func TimeItWithLogFunc(lf func(format string, v ...interface{}), skip int) func(format string, a ...interface{}) int64 {
	start := time.Now()

	pc := make([]uintptr, 1)
	runtime.Callers(skip, pc)
	f := runtime.FuncForPC(pc[0])

	return func(format string, a ...interface{}) int64 {
		tc := time.Since(start)
		exstr := fmt.Sprintf(format, a...)
		if exstr != "" {
			lf("%s (%d ms): %s", f.Name(), tc.Milliseconds(), exstr)
		}
		return tc.Milliseconds()
	}
}

func ThisFuncName() string {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}

func HttpRequest(client *http.Client, url, method, payload string, headers map[string]string) (int, []byte, error) {
	pd := strings.NewReader(payload)
	req, err := http.NewRequest(method, url, pd)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, nil, RetryErrorf(err.Error())
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}
	return res.StatusCode, body, nil
}

func HandleCrash(slog *SLOG) {
	if x := recover(); x != nil {
		slog.Errorf("Crash error: %v", x)
	}
}

func IsIpv6(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	return ip != nil && strings.Contains(ipstr, ":")
}

func Keyname(s ...string) string {
	a := []string{}
	for _, i := range s {
		if i != "" {
			a = append(a, i)
		}
	}
	return strings.Join(a, "/")
}

// Bad implementation:
//   sub-object are not copied but using as pointer instead.
// Important: map[string]string would not be recoginzed as valueMap.
// func DeepCopy(value interface{}) interface{} {
// 	if valueMap, ok := value.(map[string]interface{}); ok {
// 		newMap := make(map[string]interface{})
// 		for k, v := range valueMap {
// 			newMap[k] = DeepCopy(v)
// 		}
// 		return newMap
// 	} else if valueSlice, ok := value.([]interface{}); ok {
// 		newSlice := make([]interface{}, len(valueSlice))
// 		for k, v := range valueSlice {
// 			newSlice[k] = DeepCopy(v)
// 		}
// 		return newSlice
// 	}
// 	return value
// }

// performance: coping follow object 1000000 times cost: 2554 ms
//
//	sub := map[string]interface{}{
//		"suba": "string",
//		"subb": 12345,
//		"x":    true,
//	}
//
// function:
//
//	a, e := DeepCopy(nil)
//	a, e := DeepCopy([]string{"1", "2"})
//	a, e := DeepCopy(true)
//	a, e := DeepCopy(123)
//	a, e := DeepCopy(3.14)
//	a, e := DeepCopy(map[string]interface{}{})
//	a, e := DeepCopy("123")
func DeepCopy(value interface{}) (interface{}, error) {
	if b, err := json.Marshal(value); err != nil {
		return nil, err
	} else {
		var r interface{}
		err = json.Unmarshal(b, &r)
		return r, err
	}
}

func DeepEqual(a, b map[string]interface{}) bool {
	// data of map[string]interface{} may contain reference,
	// thus we re-un-marshal the data to a successive data.
	// what's more, after DeepCopy, int(1) != float64(1)
	var ja, jb map[string]interface{}
	if ba, err := json.Marshal(a); err != nil {
		return false
	} else {
		json.Unmarshal(ba, &ja)
	}

	if bb, err := json.Marshal(b); err != nil {
		return false
	} else {
		json.Unmarshal(bb, &jb)
	}
	return reflect.DeepEqual(ja, jb)
}

// TODO: test it with
/*
A: []interface{}{
	{"name": "customHTTPProfile"},
	{"name": "customTCPProfile"},
}
B: []interface{}{
	{"name": "customTCPProfile"},
	{"name": "customHTTPProfile"},
}

reflect.DeepEqual(SortIt(A), SortIt(B)) == true
*/
func SortIt(s *[]interface{}) []interface{} {
	tmp := map[string]interface{}{}
	ks := []string{}
	for _, v := range *s {
		bv, _ := json.Marshal(v)
		m := MD5(bv)
		copiedv, _ := DeepCopy(v)
		tmp[m] = copiedv
		ks = append(ks, m)
	}

	sort.Strings(ks)

	rlt := []interface{}{}
	for _, k := range ks {
		rlt = append(rlt, tmp[k])
	}
	return rlt
}

func MD5(v []byte) string {
	m := md5.New()
	m.Write(v)
	return hex.EncodeToString(m.Sum(nil))
}

func Diff(a, b []string) (c, d, u []string) {
	ma := map[string]string{}
	c = []string{}
	u = []string{}
	for _, n := range a {
		ma[n] = ""
	}
	for _, n := range b {
		if _, found := ma[n]; !found {
			c = append(c, n)
		} else {
			u = append(u, n)
			delete(ma, n)
		}
	}
	for k := range ma {
		d = append(d, k)
	}

	return c, d, u
}

// func JoinName(s ...string) string {
// 	a := []string{}
// 	for _, i := range s {
// 		if i != "" {
// 			a = append(a, i)
// 		}
// 	}
// 	return strings.Join(a, "_")
// }

func Contains(items []string, item string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}

func Unified(a []string) []string {
	b := map[string]bool{}
	for _, i := range a {
		b[i] = true
	}
	keys := make([]string, 0, len(b))
	for k := range b {
		keys = append(keys, k)
	}
	return keys
}

func Split(str string, size int) []string {
	a := []string{}
	l := len(str)
	if size == 0 || size >= l {
		return []string{str}
	}
	for i := 0; i < int(l/size)+1; i++ {
		next := (i + 1) * size
		if (i+1)*size > l {
			next = l
		}
		a = append(a, str[i*size:next])
	}
	return a
}

func MarshalJson(v interface{}) (map[string]interface{}, error) {
	bv, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}, err
	}

	var mv map[string]interface{}
	err = json.Unmarshal(bv, &mv)
	if err != nil {
		return map[string]interface{}{}, err
	} else {
		return mv, nil
	}
}

func UnmarshalJson(data interface{}, v interface{}) error {
	if b, err := json.Marshal(data); err != nil {
		return err
	} else {
		return json.Unmarshal(b, v)
	}
}

func MarshalNoEscaping(v interface{}) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	return b.Bytes(), err
}

func RetryErrorf(format string, v ...interface{}) error {
	return fmt.Errorf(retryMark+format, v)
}

func NeedRetry(err error) bool {
	if err == nil {
		return false
	}
	p := fmt.Sprintf("%s.*", retryMark)
	matched, e := regexp.MatchString(p, err.Error())
	if e != nil || !matched {
		return false
	} else {
		return true
	}
}

func FieldsIsExpected(fields, expected interface{}) bool {
	if reflect.TypeOf(fields) != reflect.TypeOf(expected) {
		return false
	}
	if fields == nil {
		return true
	}
	if reflect.TypeOf(fields).Kind().String() == "map" {
		for k, v := range fields.(map[string]interface{}) {
			if exp, f := expected.(map[string]interface{})[k]; !f || !reflect.DeepEqual(v, exp) {
				return false
			}
		}
		return true
	} else {
		return reflect.DeepEqual(fields, expected)
	}
}

func LogFromContext(ctx context.Context) *SLOG {
	if ctx == nil {
		return selflog
	}
	slog, ok := ctx.Value(CtxKey_Logger).(*SLOG)
	if !ok {
		return selflog
	}
	return slog
}

func RequestIdFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	reqid, ok := ctx.Value(CtxKey_RequestID).(string)
	if !ok {
		return ""
	} else {
		return reqid
	}
}
