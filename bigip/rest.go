package f5_bigip

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	utils "gitee.com/zongzw/f5-bigip-rest/utils"
)

func (bip *BIGIP) Exist(kind, name, partition, subfolder string) (*map[string]interface{}, error) {
	url := bip.URL + fmt.Sprintf("/mgmt/tm/%s", uriname(kind, refname(partition, subfolder, name)))
	method := "GET"
	payload := ""
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	var bipresp map[string]interface{}
	// logRequest(method, url, headers, payload)
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return nil, err
	}

	switch code {
	case 200:
		err = json.Unmarshal(resp, &bipresp)
		if err != nil {
			return nil, err
		} else {
			return &bipresp, nil
		}
	case 404:
		return nil, nil
	default:
		return nil, fmt.Errorf("error checking %s %s", kind, assertBigipResp20X(code, resp))
	}
}

func (bip *BIGIP) Deploy(kind, name, partition, subfolder string, body map[string]interface{}) error {
	url := bip.URL + fmt.Sprintf("/mgmt/tm/%s", kind)
	method := "POST"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	if partition != "" {
		body["partition"] = partition
	}
	if subfolder != "" {
		body["subPath"] = subfolder
	}
	body["name"] = name
	bbody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	payload := string(bbody)
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return err
	}

	return assertBigipResp20X(code, resp)
}

func (bip *BIGIP) Update(kind, name, partition, subfolder string, body map[string]interface{}) error {
	url := bip.URL + fmt.Sprintf("/mgmt/tm/%s/%s", kind, refname(partition, subfolder, name))
	method := "PATCH"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	bbody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	payload := string(bbody)
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return err
	}

	return assertBigipResp20X(code, resp)
}

func (bip *BIGIP) Delete(kind, name, partition, subfolder string) error {
	url := bip.URL + fmt.Sprintf("/mgmt/tm/%s/%s", kind, refname(partition, subfolder, name))
	method := "DELETE"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	payload := ""
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return err
	}
	return assertBigipResp20X(code, resp)
}

func (bip *BIGIP) Upload(name, content string) (string, error) {
	url := bip.URL + fmt.Sprintf("/mgmt/shared/file-transfer/uploads/%s", name)
	method := "POST"
	payload := content
	length := len(payload)
	headers := map[string]string{
		"Content-Type":   "application/octet-stream",
		"Authorization":  bip.Authorization,
		"Content-Length": fmt.Sprint(length),
		"Content-Range":  fmt.Sprintf("0-%d/%d", length-1, length),
	}

	var bipresp map[string]interface{}
	// logRequest(method, url, headers, payload)
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return "", err
	}

	switch code {
	case 200:
		err = json.Unmarshal(resp, &bipresp)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("error uploading %s", assertBigipResp20X(code, resp))
	}

	if p, f := bipresp["localFilePath"]; f {
		return p.(string), nil
	} else {
		return "", fmt.Errorf("localFilePath field not found")
	}
}

func (bip *BIGIP) All(kind string) (*map[string]interface{}, error) {
	url := bip.URL + fmt.Sprintf("/mgmt/tm/%s", kind)
	method := "GET"
	payload := ""
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	var bipresp map[string]interface{}
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return nil, err
	}

	switch code {
	case 200:
		err = json.Unmarshal(resp, &bipresp)
		if err != nil {
			return nil, err
		} else {
			return &bipresp, nil
		}
	default:
		return nil, fmt.Errorf("error retriving %s %s", kind, assertBigipResp20X(code, resp))
	}
}

func (bip *BIGIP) Tmsh(cmd string) (*map[string]interface{}, error) {
	defer utils.TimeItToPrometheus()()
	if cmd == "" {
		return &map[string]interface{}{}, nil
	}
	url := bip.URL + "/mgmt/tm/util/bash"
	method := "POST"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	body := map[string]string{
		"command":     "run",
		"utilCmdArgs": fmt.Sprintf("-c 'tmsh -c \"%s\"'", cmd),
	}
	bbody, _ := json.Marshal(body)
	payload := string(bbody)
	defer utils.TimeItTrace(&slog)("tmsh: %s %s %s", method, url, payload)
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return nil, err
	}

	var jresp map[string]interface{}
	err = json.Unmarshal(resp, &jresp)
	if err != nil {
		return nil, err
	}
	return &jresp, assertBigipResp20X(code, resp)
}

func (bip *BIGIP) Members(poolname string, partition string, subfolder string) ([]string, error) {
	mbls := []string{}
	mbsp, err := bip.Exist("ltm/pool", poolname+"/members", partition, subfolder)
	if err != nil || mbsp == nil {
		return mbls, err
	}
	mbs := *mbsp
	for _, mb := range mbs["items"].([]interface{}) {
		addr := (mb.(map[string]interface{}))["address"]
		mbls = append(mbls, addr.(string))
	}

	return mbls, nil
}

func (bip *BIGIP) Arps() (*map[string]string, error) {
	defer utils.TimeItToPrometheus()()
	arpsp, err := bip.All("net/arp")
	if err != nil {
		return nil, err
	}

	arps := map[string]string{}
	items := (*arpsp)["items"].([]interface{})
	for _, i := range items {
		mi := i.(map[string]interface{})
		arps[mi["ipAddress"].(string)] = utils.Keyname(mi["partition"].(string), mi["macAddress"].(string))
	}

	return &arps, nil
}

func (bip *BIGIP) Ndps() (*map[string]string, error) {
	defer utils.TimeItToPrometheus()()
	arpsp, err := bip.All("net/ndp")
	if err != nil {
		return nil, err
	}

	ndps := map[string]string{}
	items := (*arpsp)["items"].([]interface{})
	for _, i := range items {
		mi := i.(map[string]interface{})
		ndps[mi["ipAddress"].(string)] = utils.Keyname(mi["partition"].(string), mi["macAddress"].(string))
	}

	return &ndps, nil
}

func (bip *BIGIP) Fdbs(tunnelName string) (*map[string]string, error) {
	defer utils.TimeItToPrometheus()()

	tun := strings.ReplaceAll(tunnelName, "/", "~")
	fdbsp, err := bip.All(fmt.Sprintf("net/fdb/tunnel/%s/records", tun))
	if err != nil {
		return nil, err
	}

	fdbs := map[string]string{}
	items := (*fdbsp)["items"].([]interface{})
	for _, f := range items {
		mf := f.(map[string]interface{})
		fdbs[mf["name"].(string)] = mf["endpoint"].(string)
	}

	return &fdbs, nil
}

func (bip *BIGIP) MakeTrans() (float64, error) {
	url := bip.URL + "/mgmt/tm/transaction"
	method := "POST"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	body := map[string]interface{}{}
	bbody, _ := json.Marshal(body)
	payload := string(bbody)
	code, resp, err := httpRequest(bip.client, url, method, payload, headers)
	if err != nil {
		return 0, err
	}

	if err := assertBigipResp20X(code, resp); err != nil {
		return 0, err
	}

	var jresp map[string]interface{}
	if err := json.Unmarshal(resp, &jresp); err != nil {
		return 0, err
	} else {
		if transId, f := jresp["transId"]; !f {
			return 0, fmt.Errorf("strange.. transId not found from %v", jresp)
		} else {
			return transId.(float64), nil
		}
	}
}

func (bip *BIGIP) DeployWithTrans(rr *[]RestRequest, transId float64) (int, error) {
	defer utils.TimeItToPrometheus()()

	headersTmpl := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bip.Authorization,
	}

	count := 0
	for _, r := range *rr {
		// method
		method := r.Method
		if method == "NOPE" {
			continue
		}

		// body
		var bbody []byte
		bodyType := reflect.TypeOf(r.Body).Kind().String()
		if bodyType == "map" {
			copiedbody, err := utils.DeepCopy(r.Body)
			if err != nil {
				return 0, err
			}
			body := copiedbody.(map[string]interface{})
			if _, f := body["partition"]; !f {
				body["partition"] = r.Partition
			}
			if _, f := body["subPath"]; !f {
				body["subPath"] = r.Subfolder
			}
			mbody, err := utils.MarshalNoEscaping(body)
			if err != nil {
				return 0, fmt.Errorf("failed to marshal payload: %s, %s", r.ResName, err.Error())
			}
			bbody = mbody
		} else if bodyType == "string" {
			bbody = []byte(r.Body.(string))
		} else {
			return 0, fmt.Errorf("body type is invalid: %s", bodyType)
		}

		// url
		var url string
		switch method {
		case "POST":
			url = bip.URL + r.ResUri
		case "PATCH":
			url = bip.URL + r.ResUri + "/" + refname(r.Partition, r.Subfolder, r.ResName)
		case "DELETE":
			url = bip.URL + r.ResUri + "/" + refname(r.Partition, r.Subfolder, r.ResName)
			bbody = []byte{}
		default:
			return 0, fmt.Errorf("not support method: %s", method)
		}

		// headers
		headers := map[string]string{}
		if r.WithTrans {
			headers["X-F5-REST-Coordination-Id"] = fmt.Sprintf("%.f", transId)
		}
		for hk, hv := range headersTmpl {
			headers[hk] = fmt.Sprintf("%v", hv)
		}
		for hk, hv := range r.Headers {
			headers[hk] = fmt.Sprintf("%v", hv)
		}

		// run..
		logRequest(method, url, headers, string(bbody))
		code, resp, err := httpRequest(bip.client, url, method, string(bbody), headers)
		if err != nil {
			return 0, err
		}
		if err := assertBigipResp20X(code, resp); err != nil {
			return 0, err
		}
		if r.WithTrans {
			count += 1
		}
	}
	return count, nil
}

func (bip *BIGIP) CommitTrans(transId float64) error {
	defer utils.TimeItToPrometheus()()
	payload, _ := json.Marshal(map[string]interface{}{
		"state": "VALIDATING",
	})
	code, resp, err := httpRequest(
		bip.client,
		bip.URL+"/mgmt/tm/transaction/"+fmt.Sprintf("%.f", transId),
		"PATCH",
		string(payload),
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": bip.Authorization,
		},
	)
	if err != nil {
		return err
	}
	if err := assertBigipResp20X(code, resp); err != nil {
		return err
	}

	var jresp map[string]interface{}
	if err := json.Unmarshal(resp, &jresp); err != nil {
		return err
	} else {
		if result, f := jresp["state"]; !f {
			return fmt.Errorf("strange.. not found state from transaction response: %s", resp)
		} else {
			if result.(string) == "COMPLETED" {
				return nil
			} else {
				return fmt.Errorf("%s", resp)
			}
		}
	}
}
