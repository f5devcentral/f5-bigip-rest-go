package deployer

import (
	"fmt"

	f5_bigip "github.com/zongzw/f5-bigip-rest/bigip"
	"github.com/zongzw/f5-bigip-rest/utils"
)

func deploy(bc *f5_bigip.BIGIPContext, partition string, ocfgs, ncfgs *map[string]interface{}) error {
	defer utils.TimeItToPrometheus()()

	cmds, err := bc.GenRestRequests(partition, ocfgs, ncfgs)
	if err != nil {
		return err
	}
	return bc.DoRestRequests(cmds)
}

func HandleRequest(bc *f5_bigip.BIGIPContext, r DeployRequest) error {
	specified := r.Context.Value(CtxKey_SpecifiedBIGIP)
	slog := utils.LogFromContext(r.Context)
	if specified != nil && specified.(string) != bc.URL {
		slog.Infof("skipping bigip %s", bc.URL)
		return nil
	}

	if r.Context.Value(CtxKey_CreatePartition) != nil {
		slog.Infof("creating partition: %s", r.Partition)
		if err := bc.DeployPartition(r.Partition); err != nil {
			return fmt.Errorf("failed to deploy partition %s: %s", r.Partition, err.Error())
		}
	}
	if err := deploy(bc, r.Partition, r.From, r.To); err != nil {
		// report the error to status or ...
		return fmt.Errorf("failed to do deployment to %s: %s", bc.URL, err.Error())
	}
	if r.Context.Value(CtxKey_DeletePartition) != nil {
		slog.Infof("deleting partition: %s", r.Partition)
		if err := bc.DeletePartition(r.Partition); err != nil {
			return fmt.Errorf("failed to deploy partition %s: %s", r.Partition, err.Error())
		}
	}
	return nil
}

func Deployer(stopCh chan struct{}, bigips []*f5_bigip.BIGIP) chan DeployRequest {
	pendingDeploys := make(chan DeployRequest, 16)
	go func() {
		for {
			select {
			case <-stopCh:
				close(pendingDeploys)
				return
			case r := <-pendingDeploys:
				slog := utils.LogFromContext(r.Context)
				slog.Infof("Processing request: %s", r.Meta)
				for _, bigip := range bigips {
					bc := &f5_bigip.BIGIPContext{BIGIP: *bigip, Context: r.Context}
					if err := HandleRequest(bc, r); err != nil {
						// report status
						slog.Errorf(err.Error())
					}
				}
			}
		}
	}()
	return pendingDeploys
}
