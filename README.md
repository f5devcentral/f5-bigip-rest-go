# f5-bigip-rest

## Repository Introduction

The f5-bigip-rest repository encapsulates BIG-IP iControlRest calls in a simple and usable way. It contains two separate modules: `bigip` and `utils`

* `bigip` module can execute various BIG-IP iControlRest commands in the form of transactions, and the list of currently supported resources can be found [here](./bigip/utils.go).
* `utils` module encapsulates some necessary common objects and functions, such as logging, Prometheus monitoring, and HTTPRequest capabilities. See below for their usages.

## Module Usages

module `bigip`:

```golang

import (
    ...
	f5_bigip "gitee.com/zongzw/f5-bigip-rest/bigip"
)

func deployVirtualPool(ctx context.Context, name, partition string) error {
	bc := newBIGIPContext(ctx)

	...

	cmds, err := bc.GenRestRequests(partition, ojson, njson)
	if err != nil {
		return err
	}

    ...

	if err := bc.DeployPartition(partition); err != nil {
		return err
	}
	if err := bc.DoRestRequests(cmds); err != nil {
		return err
	}

	...

	return nil
}
```

Further, it's easy to find some detailed usages from [f5-tool-deploy-rest](https://gitee.com/zongzw/f5-tool-deploy-rest).