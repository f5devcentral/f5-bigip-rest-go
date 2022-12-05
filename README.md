# f5-bigip-rest

This is a module separated from the code repository [f5-kic](https://gitee.com/zongzw/f5-kic).

It's used to deploy BIG-IP resources via iControl REST.

See [f5-kic](https://gitee.com/zongzw/f5-kic) or [bigip-kubernetes-gateway](https://gitee.com/zongzw/bigip-kubernetes-gateway) for more details.

Also, [f5-tool-deploy-rest](https://gitee.com/zongzw/f5-tool-deploy-rest) is a simpler usage sample for reference.


```golang

import (
    ...
	f5_bigip "gitee.com/zongzw/f5-bigip-rest/bigip"
	"gitee.com/zongzw/f5-bigip-rest/utils"
)

func deployVirtualPool(ctx context.Context, name, partition string) error {
	defer utils.TimeItToPrometheus()()
	slog := utils.LogFromContext(ctx)
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