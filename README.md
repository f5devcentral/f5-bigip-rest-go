# f5-bigip-rest-go

F5 BIG-IP provides multiple kinds of configuration utilities, including xui, tmsh and iControl Rest.

This repository provides a golang library for deploying BIG-IP resources via iControl rest. 

There are 3 modules in the library.

* `bigip`

  Used to convert and execute resource requests in the transaction way. 
  
  Refer to the [example](./examples/bigip/bigip_deploy.go) for its usage. In the example, explainations are given as comments in detail.

  In the example, the resources are gathered within a JSON-format body, the body schema is:

  ```json
  {
		"<folder name>": {
			"<resource type>/<resource name>": {
				"<resource property name>": "<resource property value>",
				"...": "..."
			},
			"...": {
				"...": "..."
			}
		}
  }
  ```

  Supported resource types can be found [here](#supported-resources).

  **The caller should be clear about the very resource's properties it manipulates.** This is important to understand/use this module. 

* `utils`

  Provides necessary functions, like, *data manipulating*, *logging*, *Prometheus integrating*, and *http requesting*.

  Refer to the [example](./examples/utils/utils_sample.go) for usage. The module is widely used in `bigip` and `deployer` modules.

* `deployer`

  `deployer` is an encapsulation of `bigip`. It starts a co-routine worker waiting at a golang `chan` for executing deployment requests.
  
  The caller assembles and posts the [`DeployRequest`](./deployer/types.go) variable, and the `deployer` organizes and executes the requests.

  Refer to the [example](./examples/deployer/deployer.go) for usage.

## Supported Resources:

```shell
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
	`net/tunnels/vxlan$`,
	`net/tunnels/tunnel$`,
	`net/fdb/tunnel$`,
	`net/ndp$`,
	`net/routing/bgp$`,
	`net/self$`,
	`gtm/datacenter`,
	`gtm/server`,
	`gtm/monitor/\w+`,
	`gtm/pool/\w+`,
	`gtm/wideip`,
```

