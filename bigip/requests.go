package f5_bigip

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"gitee.com/zongzw/f5-bigip-rest/utils"
)

func (bc *BIGIPContext) DoRestRequests(rr *[]RestRequest) error {
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

	exists := map[string]map[string]interface{}{}
	partitions, err := bc.ListPartitions()
	if err != nil {
		return nil, fmt.Errorf("failed to list partitions for checking res existence: %s", err.Error())
	}
	if !utils.Contains(partitions, partition) {
		return &exists, nil
	}

	for _, kind := range kinds {
		if !(strings.HasPrefix(kind, "sys/") || strings.HasPrefix(kind, "ltm/") || strings.HasPrefix(kind, "net/")) {
			continue
		}
		exists[kind] = map[string]interface{}{}
		resp, err := bc.All(fmt.Sprintf("%s?$filter=partition+eq+%s", kind, partition))
		if err != nil {
			return nil, fmt.Errorf("failed to list '%s' of %s: %s", kind, partition, err.Error())
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

func (bc *BIGIPContext) GenRestRequests(partition string, ocfg, ncfg *map[string]interface{}) (*[]RestRequest, error) {
	defer utils.TimeItToPrometheus()()
	slog := utils.LogFromContext(bc)

	rDels := map[string][]RestRequest{}
	rCrts := map[string][]RestRequest{}

	kinds := GatherKinds(ocfg, ncfg)
	existings, err := bc.GetExistingResources(partition, kinds)
	if err != nil {
		return nil, err
	}
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
			vcmdDels = sortCmds(append(rDels["ltm/virtual"], rDels["ltm/virtual-address"]...), true)
			for i := range vcmdDels {
				vcmdDels[i].Method = "DELETE"
			}
			vcmdCrts = sortCmds(append(rCrts["ltm/virtual"], rCrts["ltm/virtual-address"]...), false)
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

	if bcmds, err := json.Marshal(cmds); err == nil {
		slog.Debugf("commands: %s", bcmds)
	}
	return &cmds, nil
}

func (bc *BIGIPContext) cfg2RestRequests(partition, operation string, cfg map[string]interface{}, exists *map[string]map[string]interface{}) (map[string][]RestRequest, error) {
	slog := utils.LogFromContext(bc)
	slog.Debugf("generating '%s' cmds for partition %s's config", operation, partition)
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

func (bc *BIGIPContext) LoadDataGroup(dgkey string) (*PersistedConfig, error) {
	dgname := "f5-kic_" + dgkey
	resp, err := bc.Exist("ltm/data-group/internal", dgname, "cis-c-tenant", "")
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	if records, f := (*resp)["records"]; !f {
		return nil, fmt.Errorf("failed to get records field")
	} else {
		pc := PersistedConfig{}
		b64as3 := ""
		b64rest := ""
		b64psmap := ""
		for _, record := range records.([]interface{}) {
			mrec := record.(map[string]interface{})
			name := mrec["name"].(string)
			if name == "cmkey" {
				pc.CmKey = string(mrec["data"].(string))
			} else if strings.HasPrefix(name, "as3") {
				b64as3 += mrec["data"].(string)
			} else if strings.HasPrefix(name, "rest") {
				b64rest += mrec["data"].(string)
			} else if strings.HasPrefix(name, "psmap") {
				b64psmap += mrec["data"].(string)
			} else {
				return nil, fmt.Errorf("invalid unknown key: %s", name)
			}
		}
		if b64as3 != "" {
			if data, err := base64.StdEncoding.DecodeString(b64as3); err != nil {
				return nil, err
			} else {
				pc.AS3 = string(data)
			}
		}

		if b64rest != "" {
			if data, err := base64.StdEncoding.DecodeString(b64rest); err != nil {
				return nil, err
			} else {
				pc.Rest = string(data)
			}
		}

		if b64psmap != "" {
			if data, err := base64.StdEncoding.DecodeString(b64psmap); err != nil {
				return nil, err
			} else {
				var psm map[string]interface{}
				err := json.Unmarshal(data, &psm)
				if err != nil {
					return nil, err
				}
				pc.PsMap = psm
			}
		}

		return &pc, nil
	}
}

func (bc *BIGIPContext) SaveDataGroup(dgkey string, pc *PersistedConfig) error {
	dgname := "f5-kic_" + dgkey
	var err error
	// failed with error:  16908375, 01020057:3: The string with more than 65535 characters cannot be stored in a message.
	blocksize := 1024
	records := []interface{}{}

	resp, err := bc.Exist("ltm/data-group/internal", dgname, "cis-c-tenant", "")
	if err != nil {
		return err
	}

	if pc.CmKey != "" {
		records = append(records, map[string]string{
			"name": "cmkey",
			"data": pc.CmKey,
		})
	}

	if pc.AS3 != "" {
		b64as3 := base64.StdEncoding.EncodeToString([]byte(pc.AS3))
		bas3s := utils.Split(b64as3, blocksize)
		for i, d := range bas3s {
			records = append(records, map[string]string{
				"name": fmt.Sprintf("as3.%d", i),
				"data": d,
			})
		}
	}

	if pc.Rest != "" {
		b64rest := base64.StdEncoding.EncodeToString([]byte(pc.Rest))
		brests := utils.Split(b64rest, blocksize)
		for i, d := range brests {
			records = append(records, map[string]string{
				"name": fmt.Sprintf("rest.%d", i),
				"data": d,
			})
		}
	}

	if len(pc.PsMap) != 0 {
		bpsm, err := json.Marshal(pc.PsMap)
		if err != nil {
			return err
		}
		b64psm := base64.StdEncoding.EncodeToString(bpsm)
		bpsms := utils.Split(b64psm, blocksize)
		for i, d := range bpsms {
			records = append(records, map[string]string{
				"name": fmt.Sprintf("psmap.%d", i),
				"data": d,
			})
		}
	}

	body := map[string]interface{}{
		"name":      dgname,
		"type":      "string",
		"partition": "cis-c-tenant",
		"records":   records,
	}

	if resp == nil {
		err = bc.Deploy("ltm/data-group/internal", dgname, "cis-c-tenant", "", body)
	} else {
		err = bc.Update("ltm/data-group/internal", dgname, "cis-c-tenant", "", body)
	}
	return err
}

func (bc *BIGIPContext) DeleteDataGroup(dgkey string) error {
	dgname := "f5-kic_" + dgkey
	var err error
	resp, err := bc.Exist("ltm/data-group/internal", dgname, "cis-c-tenant", "")
	if err != nil {
		return err
	}
	if resp != nil {
		err = bc.Delete("ltm/data-group/internal", dgname, "cis-c-tenant", "")
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
	slog := utils.LogFromContext(bc)

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
	slog := utils.LogFromContext(bc)
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

func (bc *BIGIPContext) CreateVxlanProfile(name, port string) error {
	slog := utils.LogFromContext(bc)
	var err error
	resp, err := bc.Exist("net/tunnels/vxlan", name, "Common", "")
	if err != nil {
		return err
	}
	body := map[string]interface{}{
		"name":         name,
		"floodingType": "none",
		"port":         port,
	}
	if resp == nil {
		slog.Debugf("Create vxlan profile %s here.", name)
		err = bc.Deploy("net/tunnels/vxlan", name, "Common", "", body)
	} else {
		slog.Debugf("vxlan profile %s already exists.", name)
		// err = bip.Update("net/tunnels/vxlan", name, "Common", "", body)
		return nil
	}
	return err
}

func (bc *BIGIPContext) CreateVxlanTunnel(name, key, address, profile string) error {
	slog := utils.LogFromContext(bc)
	var err error
	resp, err := bc.Exist("net/tunnels/tunnel", name, "Common", "")
	if err != nil {
		return err
	}
	body := map[string]interface{}{
		"name":         name,
		"key":          key,
		"localAddress": address,
		"profile":      profile,
	}
	if resp == nil {
		slog.Debugf("Create vxlan tunnel %s here.", name)
		err = bc.Deploy("net/tunnels/tunnel", name, "Common", "", body)
	} else {
		slog.Debugf("Update vxlan tunnel %s here.", name)
		err = bc.Update("net/tunnels/tunnel", name, "Common", "", body)
	}
	return err
}

func (bc *BIGIPContext) CreateSelf(name, address, vlan string) error {
	slog := utils.LogFromContext(bc)
	var err error
	resp, err := bc.Exist("net/self", name, "Common", "")
	if err != nil {
		return err
	}
	body := map[string]interface{}{
		"name":         name,
		"address":      address,
		"vlan":         vlan,
		"allowService": "all",
	}
	if resp == nil {
		slog.Debugf("Create selfip %s here.", name)
		err = bc.Deploy("net/self", name, "Common", "", body)
	} else {
		slog.Debugf("Update selfip %s here.", name)
		err = bc.Update("net/self", name, "Common", "", body)
	}
	return err
}
