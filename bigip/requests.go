package f5_bigip

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/f5devcentral/f5-bigip-rest-go/utils"
)

func (bc *BIGIPContext) DoRestRequests(rr *[]RestRequest) error {
	if rr == nil || len(*rr) == 0 {
		slog := utils.LogFromContext(bc.Context)
		slog.Debugf("empty rest requests, skip deploying")
		return nil
	}
	if transId, err := bc.MakeTrans(); err != nil {
		return err
	} else {
		if count, err := bc.DeployWithTrans(rr, transId); err != nil || count == 0 {
			return err
		} else {
			return bc.CommitTrans(transId)
		}
	}
}

func (bc *BIGIPContext) constructFolder(name, partition string) RestRequest {
	kind := "sys/folder"
	return RestRequest{
		Method: "NOPE",
		Body: map[string]interface{}{
			"name":      name,
			"partition": partition,
		},
		ResUri:    "/mgmt/tm/" + kind,
		Kind:      kind,
		ResName:   name,
		Partition: partition,
		Subfolder: "",
		WithTrans: true,
	}
}

func (bc *BIGIPContext) constructLTMRes(kind, name, partition, subfolder string, body interface{}) RestRequest {
	return RestRequest{
		Method:    "NOPE",
		Headers:   map[string]interface{}{},
		Body:      body,
		ResUri:    "/mgmt/tm/" + kind,
		Kind:      kind,
		ResName:   name,
		Partition: partition,
		Subfolder: subfolder,
		WithTrans: true,
	}
}

func (bc *BIGIPContext) constructGTMRes(kind, name, partition, subfolder string, body interface{}) RestRequest {
	return RestRequest{
		Method:    "NOPE",
		Headers:   map[string]interface{}{},
		Body:      body,
		ResUri:    "/mgmt/tm/" + kind,
		Kind:      kind,
		ResName:   name,
		Partition: partition,
		Subfolder: subfolder,
		WithTrans: true,
	}
}

func (bc *BIGIPContext) constructNetRes(kind, name, partition, subfolder string, body interface{}) RestRequest {
	return RestRequest{
		Method:    "NOPE",
		Headers:   map[string]interface{}{},
		Body:      body,
		ResUri:    "/mgmt/tm/" + kind,
		Kind:      kind,
		ResName:   name,
		Partition: partition,
		Subfolder: subfolder,
		WithTrans: true,
	}
}

func (bc *BIGIPContext) constructSysRes(kind, name, partition, subfolder string, body interface{}) RestRequest {
	return RestRequest{
		Method:    "NOPE",
		Body:      body,
		Headers:   map[string]interface{}{},
		ResUri:    "/mgmt/tm/" + kind,
		Kind:      kind,
		ResName:   name,
		Partition: partition,
		Subfolder: subfolder,
		WithTrans: true,
	}
}

func (bc *BIGIPContext) constructSharedRes(kind, name, partition, subfolder string, body interface{}, operation string) (RestRequest, error) {
	r := RestRequest{}

	switch kind {
	case "shared/file-transfer/uploads":
		if operation == "deploy" {
			rawbody := body.(map[string]interface{})["content"].(string)
			size := len(rawbody)
			r = RestRequest{
				Method: "POST",
				Body:   rawbody,
				ResUri: "/mgmt/shared/file-transfer/uploads/" + name,
				Headers: map[string]interface{}{
					"Content-Type":   "application/octet-stream",
					"Content-Length": fmt.Sprintf("%d", size),
					"Content-Range":  fmt.Sprintf("0-%d/%d", size-1, size),
				},
				Partition: partition,
				Subfolder: subfolder,
				ResName:   name,
				Kind:      kind,
				WithTrans: false,
			}
		} else if operation == "delete" {
			// the uploaded file would be removed automatically by BIG-IP,
			// we needn't to handle it.
			r = RestRequest{
				ScheduleIt: "never",
				Method:     "POST",
				Body: map[string]interface{}{
					"command":     "run",
					"utilCmdArgs": fmt.Sprintf("-c 'rm -f /var/config/rest/downloads/%s'", name),
				},
				ResUri:    "/mgmt/tm/util/bash",
				Partition: partition,
				Subfolder: subfolder,
				ResName:   name,
				Kind:      kind,
				WithTrans: false,
			}
		}

	default:
		return r, fmt.Errorf("not supported kind %s", kind)
	}

	return r, nil
}

func (bc *BIGIPContext) GetExistingResources(partition string, kinds []string) (*map[string]map[string]interface{}, error) {
	defer utils.TimeItToPrometheus()()
	slog := utils.LogFromContext(bc.Context)

	exists := map[string]map[string]interface{}{}

	regxNotfound := regexp.MustCompile(`The requested \w+ (.*) was not found.`)
	regxNoFolder := regexp.MustCompile(`The requested folder (.*) was not found.`)

	for _, kind := range kinds {
		if !KindIsSupported(kind) {
			slog.Errorf("kind %s not support, yet", kind)
			continue
		}
		exists[kind] = map[string]interface{}{}
		resp, err := bc.All(fmt.Sprintf("%s?$filter=partition+eq+%s", kind, partition))
		if err != nil {
			if regxNoFolder.MatchString(err.Error()) {
				return &exists, nil
			} else if regxNotfound.MatchString(err.Error()) {
				continue
			} else {
				return nil, fmt.Errorf("failed to list '%s' of %s: %s", kind, partition, err.Error())
			}
		}

		if items, ok := (*resp)["items"]; !ok {
			return nil, fmt.Errorf("failed to get items from response")
		} else {
			for _, item := range items.([]interface{}) {
				props := item.(map[string]interface{})
				p, f, n := partition, "", props["name"].(string)
				if ff, ok := props["subPath"]; ok {
					f = ff.(string)
				}
				exists[kind][utils.Keyname(p, f, n)] = props
			}
		}
	}
	return &exists, nil
}

// GenRestRequests generate a list of rest requests, each item is type of RestRequest
// GenRestRequests will compare the passed ocfg and ncfg, and in addition the actual states
// got from BIG-IP, and concludes into a list of RestRequests indicating
// which resource needs to be POST, PATCH or DELETE, and in which order.
// The generated []RestRequest will be used by DoRestRequests function for execution in trasaction mode.
// ocfg, ncfg format:
//
//	{
//		"<folder name>": {
//			"[ltm|net|...]/<resource type>/<resource name>": {
//				"resource property key": "resource property value",
//				"...": "..."
//			}
//		},
//		"...": "..."
//	}
//
// The data transformation is:
//
//	{ocfgs}             	{ncfgs}
//
// {typed-[rDels]}       {typed-[rCrts]}
//
//	[c]         [u]        [d]
//
//	        Existings
//
// u       c   u       c  d      n/a
//
//	[sorted-rrs]
func (bc *BIGIPContext) GenRestRequests(partition string, ocfg, ncfg *map[string]interface{}, existings *map[string]map[string]interface{}) (*[]RestRequest, error) {
	defer utils.TimeItToPrometheus()()
	slog := utils.LogFromContext(bc.Context)

	rDels := map[string][]RestRequest{}
	rCrts := map[string][]RestRequest{}

	if ocfg != nil {
		var err error
		if rDels, err = bc.cfg2RestRequests(partition, "delete", *ocfg, existings); err != nil {
			return &[]RestRequest{}, err
		}
	}
	if ncfg != nil {
		var err error
		if rCrts, err = bc.cfg2RestRequests(partition, "deploy", *ncfg, existings); err != nil {
			return &[]RestRequest{}, err
		}
	}

	vcmdDels, vcmdCrts := []RestRequest{}, []RestRequest{}
	// if there were virtual-address change ...
	// this 'if' block is used to handle the case of: virtual-address's name is not IP addr which
	// is deployed via AS3 ever before.
	// i.e.   "app_svc_vip": {
	// 			"class": "Service_Address",
	// 			"virtualAddress": "172.16.142.112",
	// 			"arpEnabled": true
	// 		  },
	// this case may happen in migration process
	if virtualAddressNameDismatched(append(rDels["ltm/virtual-address"], rCrts["ltm/virtual-address"]...)) {
		rDelVs := map[string][]RestRequest{
			"ltm/virtual":         rDels["ltm/virtual"],
			"ltm/virtual-address": rDels["ltm/virtual-address"],
		}
		rCrtVs := map[string][]RestRequest{
			"ltm/virtual":         rCrts["ltm/virtual"],
			"ltm/virtual-address": rCrts["ltm/virtual-address"],
		}
		cvl, dvl, uvl := sweepCmds(rDelVs, rCrtVs, existings)
		if len(cvl)+len(dvl)+len(uvl) != 0 {
			delete(rDels, "ltm/virtual")
			delete(rDels, "ltm/virtual-address")
			delete(rCrts, "ltm/virtual")
			delete(rCrts, "ltm/virtual-address")
			vcmdDels = sortCmds(append(rDelVs["ltm/virtual"], rDelVs["ltm/virtual-address"]...), true)
			for i := range vcmdDels {
				vcmdDels[i].Method = "DELETE"
			}
			vcmdCrts = sortCmds(append(rCrtVs["ltm/virtual"], rCrtVs["ltm/virtual-address"]...), false)
			for i := range vcmdCrts {
				vcmdCrts[i].Method = "POST"
			}
		}
	}

	cl, dl, ul := sweepCmds(rDels, rCrts, existings)
	cmds := layoutCmds(cl, dl, ul)
	cmds = append(cmds, vcmdDels...)
	cmds = append(cmds, vcmdCrts...)

	// if there is virtual-address change...

	// TODO: handle [{"ResName":"120.0.0.0%!"(MISSING), issue.
	if bcmds, err := json.Marshal(cmds); err == nil {
		slog.Tracef("commands: %s", bcmds)
	}
	return &cmds, nil
}

func (bc *BIGIPContext) cfg2RestRequests(partition, operation string, cfg map[string]interface{}, exists *map[string]map[string]interface{}) (map[string][]RestRequest, error) {
	slog := utils.LogFromContext(bc.Context)
	slog.Tracef("generating '%s' cmds for partition %s's config", operation, partition)
	rrs := map[string][]RestRequest{}

	for fn, ress := range cfg {
		if fn != "" {
			rSubfolder := bc.constructFolder(fn, partition)
			rSubfolder.Method = opr2method(operation, nil != getFromExists("sys/folder", partition, "", fn, exists))
			if _, f := rrs["sys/folder"]; !f {
				rrs["sys/folder"] = []RestRequest{}
			}
			rrs["sys/folder"] = append(rrs["sys/folder"], rSubfolder)
		}

		for tn, body := range ress.(map[string]interface{}) {
			tnarr := strings.Split(tn, "/")
			t := strings.Join(tnarr[0:len(tnarr)-1], "/")
			rootKind := tnarr[0]
			n := tnarr[len(tnarr)-1]
			var r RestRequest
			var err error = nil
			switch rootKind {
			case "ltm":
				r = bc.constructLTMRes(t, n, partition, fn, body)
				r.Method = opr2method(operation, nil != getFromExists(t, partition, fn, n, exists))
			case "gtm":
				r = bc.constructGTMRes(t, n, partition, fn, body)
				r.Method = opr2method(operation, nil != getFromExists(t, partition, fn, n, exists))
			case "net":
				r = bc.constructNetRes(t, n, partition, fn, body)
				r.Method = opr2method(operation, nil != getFromExists(t, partition, fn, n, exists))
			case "sys":
				r = bc.constructSysRes(t, n, partition, fn, body)
				r.Method = opr2method(operation, nil != getFromExists(t, partition, fn, n, exists))
			case "shared":
				r, err = bc.constructSharedRes(t, n, partition, fn, body, operation)
			default:
				return rrs, fmt.Errorf("not support root kind: %s", rootKind)
			}
			if err != nil {
				return rrs, err
			} else {
				if _, f := rrs[t]; !f {
					rrs[t] = []RestRequest{}
				}
				if r.ScheduleIt != "" {
					// TODO: add it to resSyncer
				} else {
					rrs[t] = append(rrs[t], r)
				}
			}
		}
	}
	return rrs, nil
}

// DeployPartition create the specified partition if not exists on BIG-IP
func (bc *BIGIPContext) DeployPartition(name string) error {
	if name == "Common" {
		return nil
	}
	pobj, err := bc.Exist("sys/folder", "", name, "")
	if err != nil {
		return err
	}

	if pobj == nil {
		return bc.Deploy("sys/folder", name, "/", "", map[string]interface{}{})
	}
	return nil
}

// DeletePartition delete the specified partition if exists on BIG-IP
func (bc *BIGIPContext) DeletePartition(name string) error {
	if name == "Common" {
		return nil
	}
	if f, err := bc.Exist("sys/folder", "", name, ""); err != nil {
		return err
	} else if f == nil {
		return nil
	}
	return bc.Delete("sys/folder", name, "", "")
}

func (bc *BIGIPContext) LoadDataGroup(dgname, partition string) ([]byte, error) {
	resp, err := bc.Exist("ltm/data-group/internal", dgname, partition, "")
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	if records, f := (*resp)["records"]; !f {
		return nil, fmt.Errorf("failed to get records field")
	} else if (*resp)["type"] != "string" {
		return nil, fmt.Errorf("data group type is not string")
	} else {
		b64bytes := []byte{}
		for _, record := range records.([]interface{}) {
			data := record.(map[string]interface{})["data"].(string)
			b64bytes = append(b64bytes, []byte(data)...)
		}

		return base64.StdEncoding.DecodeString(string(b64bytes))
	}
}

func (bc *BIGIPContext) SaveDataGroup(dgname string, partition string, bytes []byte) error {
	var err error
	records := []interface{}{}

	// failed with error:  16908375, 01020057:3: The string with more than 65535 characters cannot be stored in a message.
	resp, err := bc.Exist("ltm/data-group/internal", dgname, partition, "")
	if err != nil {
		return err
	}

	b64bytes := base64.StdEncoding.EncodeToString(bytes)
	u := 1024
	c := int(len(b64bytes) / u)
	m := int(len(b64bytes) % u)
	for i := 0; i < c; i++ {
		records = append(records, map[string]string{
			"name": fmt.Sprintf("%d", i),
			"data": string(b64bytes[i*u : (i+1)*u]),
		})
	}
	if m > 0 {
		records = append(records, map[string]string{
			"name": fmt.Sprintf("%d", c),
			"data": string(b64bytes[c*u:]),
		})
	}

	body := map[string]interface{}{
		"name":      dgname,
		"type":      "string",
		"partition": partition,
		"records":   records,
	}

	if resp == nil {
		err = bc.Deploy("ltm/data-group/internal", dgname, partition, "", body)
	} else {
		err = bc.Update("ltm/data-group/internal", dgname, partition, "", body)
	}
	return err
}

func (bc *BIGIPContext) DeleteDataGroup(dgname, partition string) error {
	var err error
	resp, err := bc.Exist("ltm/data-group/internal", dgname, partition, "")
	if err != nil {
		return err
	}
	if resp != nil {
		err = bc.Delete("ltm/data-group/internal", dgname, partition, "")
	}
	return err
}

func (bc *BIGIPContext) ListPartitions() ([]string, error) {
	partitions := []string{}
	resp, err := bc.All("sys/folder")
	if err != nil {
		return partitions, fmt.Errorf("failed to list partitions: %s", err.Error())
	}

	if items, ok := (*resp)["items"]; !ok {
		return partitions, fmt.Errorf("failed to get items from response")
	} else {
		for _, item := range items.([]interface{}) {
			props := item.(map[string]interface{})
			if fullPath, f := props["fullPath"].(string); f {
				paths := strings.Split(fullPath, "/")
				if len(paths) == 2 && paths[1] != "" {
					partitions = append(partitions, paths[1])
				}
			}
		}
	}
	return utils.Unified(partitions), nil
}

func (bc *BIGIPContext) SaveSysConfig(partitions []string) error {
	slog := utils.LogFromContext(bc.Context)

	cmd := "save sys config"
	if len(partitions) > 0 {
		cmd += "partitions { "

		for _, p := range partitions {
			cmd += p + " "
		}
		cmd += "}"
	}

	resp, err := bc.Tmsh(cmd)
	if err != nil {
		return err
	}
	if (*resp)["commandResult"] != nil {
		slog.Warnf("command %s: %v", cmd, (*resp)["commandResult"])
	}
	return nil
}

func (bc *BIGIPContext) ModifyDbValue(name, value string) error {
	slog := utils.LogFromContext(bc.Context)
	// modify sys db tmrouted.tmos.routing value enable
	cmd := "modify sys db "
	cmd += name
	cmd += " value "
	cmd += value
	slog.Debugf("cmd is: %s", cmd)

	resp, err := bc.Tmsh(cmd)

	if err != nil {
		return err
	}

	if (*resp)["commandResult"] != nil {
		slog.Warnf("command %s: %v", cmd, (*resp)["commandResult"])
	}
	return nil
}
