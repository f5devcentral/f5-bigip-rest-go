package f5_bigip

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	utils "gitee.com/zongzw/f5-bigip-rest/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	ResOrder = []string{
		`sys/folder`,
		`shared/file-transfer/uploads`,
		`sys/file/ssl-(cert|key)`,
		`ltm/monitor/\w+`,
		`ltm/node`,
		`ltm/pool`,
		`ltm/snat-translation`,
		`ltm/snatpool`,
		`ltm/profile/\w+`,
		`ltm/persistence/\w+`,
		`ltm/snat$`,
		`ltm/rule$`,
		`ltm/virtual-address`,
		`ltm/virtual$`,
		`net/arp$`,
		`net/fdb/tunnel$`,
		`net/ndp$`,
		`net/routing/bgp$`,
	}
	BIGIPiControlTimeCostTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bigip_icontrol_timecost_total",
			Help: "time cost(in milliseconds) of bigip icontrol rest api calls",
		},
		[]string{"method", "url"},
	)

	BIGIPiControlTimeCostCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bigip_icontrol_timecost_count",
			Help: "total number of bigip icontrol rest api calls",
		},
		[]string{"method", "url"},
	)
}

func Initialize(url, user, password, logLevel string) *BIGIP {
	return setupBIGIP(url, user, password)
}

func setupBIGIP(url, user, password string) *BIGIP {
	bauth := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	bip := BIGIP{
		URL:           url,
		Authorization: bauth,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Timeout: 60 * time.Second,
		},
	}

	bc := &BIGIPContext{
		bip,
		context.TODO(),
	}
	sysinfo, err := bc.All("sys/version")
	if err != nil {
		panic(fmt.Errorf("BIGIP %s is unavailable: err %s, quit", bip.URL, err.Error()))
	} else if sysinfo == nil {
		panic(fmt.Errorf("BIGIP %s is unavailable: %s, quit", bip.URL, "cannot get sys info"))
	} else {
		bip.Version, err = bigipVersion(*sysinfo)
		if err != nil {
			panic(err)
		}
	}

	if err := bc.DeployPartition("cis-c-tenant"); err != nil {
		panic(err)
	}
	return &bip
}

func assertBigipResp20X(statusCode int, resp []byte) error {
	sresp := string(resp)
	switch statusCode {
	case 401:
		return utils.RetryErrorf("%d, %s", statusCode, sresp)
	case 503:
		return utils.RetryErrorf("%d, %s", statusCode, sresp)
	case 500:
		return utils.RetryErrorf("%d, %s", statusCode, sresp)
	case 404:
		for _, p := range []string{
			".*URI path .* not registered.*",
			".*Public URI path not registered: .*",
		} {
			if matched, err := regexp.Match(p, resp); err == nil && matched {
				return utils.RetryErrorf("%d, %s", statusCode, sresp)
			}
		}
		return fmt.Errorf("%d, %s", statusCode, sresp)
	default:
		if int(statusCode/200) != 1 {
			return fmt.Errorf("%d, %s", statusCode, sresp)
		} else {
			return nil
		}
	}

	// kinds of error statuses from BIG-IP

	// restart restjavad
	// 503: long html response..: Configuration Utility restarting...
	// 404: {"code":404,"message":"URI path /mgmt/tm/ltm/pool/?Common?my-pool not registered.  Please verify URI is supported and wait for /available suffix to be responsive.","restOperationId":41,"kind":":resterrorresponse"}
	// 404: {"code":404,"message":"Public URI path not registered: /tm/ltm/pool/?Common?my-pool","referer":"10.250.64.100","restOperationId":39168,"kind":":resterrorresponse"

	// restart mcpd
	// 404: {"code":404,"message":"01020036:3: The requested Pool (/Common/my-pool) was not found.","errorStack":[],"apiError":3}
	// 500: {"code":500,"message":"The connection to mcpd has been lost, try again.","errorStack":[],"apiError":32768001}

	// occasional
	// 401, {"code":401,"message":"Authorization failed: no user authentication header or token detected. Uri:http://localhost:8100/mgmt/tm/ltm/virtual Referrer:10.145.74.44 Sender:10.145.74.44","referer":"10.145.74.44","restOperationId":7945916,"kind":":resterrorresponse"}

	// gui error but no impact to restapi
	// https://10.250.118.253:8443/tmui/login.jsp?msgcode=2&
}

func refname(partition, subfolder, name string) string {
	l := []string{}
	for _, x := range []string{partition, subfolder, name} {
		if x != "" {
			l = append(l, x)
		}
	}
	rn := strings.Join(l, "~")
	if rn == "" {
		return rn
	} else {
		return "~" + rn
	}
}

func bigipVersion(sysinfo map[string]interface{}) (string, error) {
	if entries, f := sysinfo["entries"]; f {
		if version0, f := entries.(map[string]interface{})["https://localhost/mgmt/tm/sys/version/0"]; f {
			if nestedStats, f := version0.(map[string]interface{})["nestedStats"]; f {
				if entries, f := nestedStats.(map[string]interface{})["entries"]; f {
					if version, f := entries.(map[string]interface{})["Version"]; f {
						if description, f := version.(map[string]interface{})["description"]; f {
							return description.(string), nil
						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("entries not found")
}

func logRequest(ctx context.Context, method, url string, headers map[string]string, body string) {
	slog := utils.LogFromContext(ctx)

	uris := strings.Split(url, "/mgmt")
	if len(uris) >= 2 {
		uri := strings.Join(uris[1:], "/mgmt")
		slog.Debugf("#### %s %s", method, uri)
	} else {
		slog.Debugf("#### %s %s", method, url)
	}
	slog.Debugf("%s %s", method, url)
	for k, v := range headers {
		slog.Debugf("%s: %s", k, v)
	}
	slog.Debugf("%s", body)
	slog.Debugf("")
}

func uriname(s ...string) string {
	a := []string{}
	for _, i := range s {
		if i != "" {
			a = append(a, i)
		}
	}
	return strings.Join(a, "/")
}

func opr2method(operation string, exist bool) string {
	if operation == "deploy" {
		if exist {
			return "PATCH"
		} else {
			return "POST"
		}
	} else {
		if exist {
			return "DELETE"
		} else {
			return "NOPE"
		}
	}
}

// func sortRestRequests(rrmap map[string][]RestRequest, operation string) []RestRequest {
// 	rtn := []RestRequest{}
// 	orderDeploy := ResOrder
// 	orderDelete := []string{}
// 	for i := len(orderDeploy) - 1; i >= 0; i-- {
// 		orderDelete = append(orderDelete, orderDeploy[i])
// 	}
// 	var order []string
// 	if operation == "deploy" {
// 		order = orderDeploy
// 	} else if operation == "delete" {
// 		order = orderDelete
// 	}
// 	for _, t := range order {
// 		rex := regexp.MustCompile(t)
// 		for k, rr := range rrmap {
// 			if rex.MatchString(k) {
// 				rtn = append(rtn, rr...)
// 				break
// 			}
// 		}
// 	}

//		return rtn
//	}

func sortCmds(unsorted []RestRequest, reversed bool) []RestRequest {
	order := ResOrder
	if reversed {
		order = []string{}
		for i := len(ResOrder) - 1; i >= 0; i-- {
			order = append(order, ResOrder[i])
		}
	}
	sorted := []RestRequest{}
	m := map[string][]RestRequest{}
	for _, r := range unsorted {
		if _, f := m[r.Kind]; !f {
			m[r.Kind] = []RestRequest{}
		}
		m[r.Kind] = append(m[r.Kind], r)
	}
	for _, krex := range order {
		for k, rs := range m {
			if matched, err := regexp.MatchString(krex, k); err == nil && matched {
				sorted = append(sorted, rs...)
			}
		}
	}
	return sorted
}

func httpRequest(ctx context.Context, client *http.Client, url, method, payload string, headers map[string]string) (int, []byte, error) {
	slog := utils.LogFromContext(ctx)

	tf := utils.TimeItTrace(slog)
	defer func() {
		rec := url
		tnarr := strings.Split(rec, "?")
		tnarr = strings.Split(tnarr[0], "/mgmt")
		if len(tnarr) >= 2 {
			uri := "/mgmt" + strings.Join(tnarr[1:], "/mgmt")
			tnarr = strings.Split(uri, "/")
			r := ""
			for _, n := range tnarr {
				if n != "" && rune('a') <= rune(n[0]) && rune('z') >= rune(n[0]) {
					r += "/" + n
				}
			}
			if r != "" {
				rec = r
			}
		}
		tc := float64(tf("%s %s", method, url))
		BIGIPiControlTimeCostCount.WithLabelValues(method, rec).Inc()
		BIGIPiControlTimeCostTotal.WithLabelValues(method, rec).Add(tc)
	}()

	return utils.HttpRequest(client, url, method, payload, headers)
}

func GatherKinds(ocfg, ncfg *map[string]interface{}) []string {
	kinds := []string{
		"sys/folder",
	}
	if ocfg != nil {
		for _, ress := range *ocfg {
			for tn := range ress.(map[string]interface{}) {
				tnarr := strings.Split(tn, "/")
				t := strings.Join(tnarr[0:len(tnarr)-1], "/")
				kinds = append(kinds, t)
			}
		}
	}
	if ncfg != nil {
		for _, ress := range *ncfg {
			for tn := range ress.(map[string]interface{}) {
				tnarr := strings.Split(tn, "/")
				t := strings.Join(tnarr[0:len(tnarr)-1], "/")
				kinds = append(kinds, t)
			}
		}
	}
	kinds = utils.Unified(kinds)

	return kinds
}

func getFromExists(kind, partition, subfolder, name string, exists *map[string]map[string]interface{}) *interface{} {
	if exists == nil {
		return nil
	}
	if res, kf := (*exists)[kind]; kf {
		pfn := utils.Keyname(partition, subfolder, name)
		if rlt, rf := res[pfn]; rf {
			return &rlt
		}
	}
	return nil
}

func virtualAddressNameDismatched(rr []RestRequest) bool {
	for _, r := range rr {
		if r.ResUri == "/mgmt/tm/ltm/virtual-address" {
			if jbody, ok := r.Body.(map[string]interface{}); ok && jbody["address"] != r.ResName {
				return true
			}
		}
	}
	return false
}

func sweepCmds(dels, crts map[string][]RestRequest, existings *map[string]map[string]interface{}) ([]RestRequest, []RestRequest, []RestRequest) {
	c, d, u := []RestRequest{}, []RestRequest{}, []RestRequest{}

	splitCmds := func(drs, crs []RestRequest) {
		dl := []string{}
		dm := map[string]RestRequest{}
		for _, dr := range drs {
			pfn := utils.Keyname(dr.Partition, dr.Subfolder, dr.ResName)
			dl = append(dl, pfn)
			dm[pfn] = dr
		}
		cl := []string{}
		cm := map[string]RestRequest{}
		for _, cr := range crs {
			pfn := utils.Keyname(cr.Partition, cr.Subfolder, cr.ResName)
			cl = append(cl, pfn)
			cm[pfn] = cr
		}
		sc, sd, su := utils.Diff(dl, cl)
		for _, s := range sc {
			c = append(c, cm[s])
		}
		for _, s := range sd {
			d = append(d, dm[s])
		}
		for _, s := range su {
			u = append(u, cm[s])
		}
	}

	for k, drs := range dels {
		if _, f := crts[k]; !f {
			d = append(d, drs...)
		}
	}
	for k, crs := range crts {
		if _, f := dels[k]; !f {
			c = append(c, crs...)
		}
	}
	for k, crs := range crts {
		if drs, f := dels[k]; f {
			splitCmds(drs, crs)
		}
	}

	cc, dd, uu := []RestRequest{}, []RestRequest{}, []RestRequest{}

	for _, r := range c {
		b := getFromExists(r.Kind, r.Partition, r.Subfolder, r.ResName, existings)
		if b == nil {
			r.Method = "POST"
			cc = append(cc, r)
		} else {
			if !utils.FieldsIsExpected(r.Body, *b) {
				r.Method = "PATCH"
				uu = append(uu, r)
			}
		}
	}
	for _, r := range d {
		b := getFromExists(r.Kind, r.Partition, r.Subfolder, r.ResName, existings)
		if b == nil {
			r.Method = "NOPE"
		} else {
			r.Method = "DELETE"
			dd = append(dd, r)
		}
	}
	for _, r := range u {
		b := getFromExists(r.Kind, r.Partition, r.Subfolder, r.ResName, existings)
		if b == nil {
			r.Method = "POST"
			cc = append(cc, r)
		} else {
			if !utils.FieldsIsExpected(r.Body, *b) {
				r.Method = "PATCH"
				uu = append(uu, r)
			}
		}
	}

	return cc, dd, uu
}

func layoutCmds(c, d, u []RestRequest) []RestRequest {
	cmds := []RestRequest{}

	cc := sortCmds(c, false)
	dd := sortCmds(d, true)
	uu := sortCmds(u, false)

	cmds = append(cmds, cc...)
	cmds = append(cmds, uu...)
	cmds = append(cmds, dd...)

	return cmds
}
