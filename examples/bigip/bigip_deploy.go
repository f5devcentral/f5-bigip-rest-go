package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

	// The format of the content is JSON, as native as:
	/*
		{
			"<folder name to place the resources>": {
				"<resource type starting with 'ltm/net/sys/gtm/..'>/<resource name>": {
					"iControlRest key":  "and value",
					"can be found from": "https://clouddocs.f5.com/api/icontrol-rest/"
				}
			}
		}
	*/
	configs := fmt.Sprintf(`
		{
			"service_name_app": {
				"ltm/monitor/http/service_name_monitor": {
					"adaptive": "disabled",
					"adaptiveDivergenceMilliseconds": 500,
					"adaptiveDivergenceType": "relative",
					"adaptiveDivergenceValue": 100,
					"adaptiveLimit": 1000,
					"adaptiveSamplingTimespan": 180,
					"interval": 5,
					"ipDscp": 0,
					"name": "service_name_monitor",
					"recv": "200 OK",
					"recvDisable": "",
					"reverse": "disabled",
					"send": "GET / HTTP/1.0\\r\\nHost:vhost.example.com\\r\\n\\r\\n",
					"targetAddress": "",
					"targetPort": 0,
					"timeUntilUp": 0,
					"timeout": 16,
					"transparent": "disabled",
					"upInterval": 0
				},
				"ltm/pool/service_name_pool": {
					"allowNat": "yes",
					"allowSnat": "yes",
					"loadBalancingMode": "least-connections-member",
					"members": [],
					"minActiveMembers": 1,
					"minimumMonitors": 1,
					"monitor": "/%s/service_name_app/service_name_monitor",
					"name": "service_name_pool",
					"reselectTries": 0,
					"serviceDownAction": "none",
					"slowRampTime": 10
				},
				"ltm/profile/http/service_name_httpprofile": {
					"acceptXff": "disabled",
					"hstsIncludeSubdomains": true,
					"hstsInsert": false,
					"hstsPeriod": 7862400,
					"hstsPreload": false,
					"insertXforwardedFor": "enabled",
					"knownMethods": [
						"CONNECT",
						"DELETE",
						"GET",
						"HEAD",
						"LOCK",
						"OPTIONS",
						"POST",
						"PROPFIND",
						"PUT",
						"TRACE",
						"UNLOCK"
					],
					"maxHeaderCount": 64,
					"maxHeaderSize": 32768,
					"maxRequests": 0,
					"name": "service_name_httpprofile",
					"oneconnectTransformations": "enabled",
					"pipelineAction": "allow",
					"proxyConnectEnabled": false,
					"proxyType": "reverse",
					"redirectRewrite": "none",
					"serverAgentName": "BigIP",
					"truncatedRedirects": false,
					"unknownMethodAction": "allow",
					"viaRequest": "remove",
					"viaResponse": "remove",
					"webSocketMasking": "unmask",
					"webSocketsEnabled": false
				},
				"ltm/profile/one-connect/service_name_oneconnectprofile": {
					"idleTimeoutOverride": 0,
					"limitType": "none",
					"maxAge": 86400,
					"maxReuse": 5,
					"maxSize": 10000,
					"name": "service_name_oneconnectprofile",
					"sharePools": "disabled",
					"sourceMask": "255.255.255.255"
				},
				"ltm/snatpool/service_name_vs_self_0": {
					"members": [
						"197.14.222.12"
					],
					"name": "service_name_vs_self_0"
				},
				"ltm/virtual/service_name_vs_0": {
					"addressStatus": "yes",
					"connectionLimit": 0,
					"destination": "197.14.222.12:80",
					"enable": true,
					"httpMrfRoutingEnabled": false,
					"ipProtocol": "tcp",
					"lastHop": "default",
					"mirror": "disabled",
					"name": "service_name_vs_0",
					"nat64": "disabled",
					"persist": [
						{
							"name": "cookie"
						}
					],
					"pool": "/%s/service_name_app/service_name_pool",
					"profileTCP": "normal",
					"profiles": [
						{
							"name": "/%s/service_name_app/service_name_httpprofile"
						},
						{
							"name": "/%s/service_name_app/service_name_oneconnectprofile"
						}
					],
					"rateLimit": 0,
					"serviceDownImmediateAction": "none",
					"shareAddresses": false,
					"sourceAddressTranslation": {
						"pool": "service_name_vs_self_0",
						"type": "snat"
					},
					"sourcePort": "preserve",
					"translateAddress": "enabled",
					"translatePort": "enabled",
					"virtualType": "standard"
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

	// reversely, we exchange the position of ncfgs and nil, a set of deletion RestRequest will be generated.
	if cmds, err := bc.GenRestRequests(partition, &ncfgs, nil); err != nil {
		fmt.Printf("failed to generate rest requests for deleting: %s\n", err.Error())
		panic(err)
	} else {
		// execute the deletion for resources.
		if err := bc.DoRestRequests(cmds); err != nil {
			fmt.Printf("failed to deploy with rest requests: %s\n", err.Error())
			panic(err)
		} else {
			fmt.Println("deleted requests.")
			kinds := []string{"ltm/virtual", "ltm/pool", "ltm/monitor/http"}
			// verify the resources are deleted on the BIG-IP
			if existings, err := bc.GetExistingResources(partition, kinds); err != nil {
				fmt.Printf("failed to get existing resources of %s: %s\n", kinds, err.Error())
				panic(err)
			} else {
				b, _ := json.MarshalIndent(existings, "", "  ")
				fmt.Println(string(b))
			}
		}
	}

	// Delete the partition
	if err := bc.DeletePartition(partition); err != nil {
		fmt.Printf("failed to delete partition: %s: %s\n", partition, err.Error())
		panic(err)
	}
}
