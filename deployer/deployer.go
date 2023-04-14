package deployer

import (
	"fmt"
	"sync"

	f5_bigip "github.com/f5devcentral/f5-bigip-rest/bigip"
	"github.com/f5devcentral/f5-bigip-rest/utils"
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

func Deployer(stopCh chan struct{}, bigips []*f5_bigip.BIGIP) (chan DeployRequest, *DeployResponses) {
	pendingDeploys := make(chan DeployRequest, 16)
	doneDeploys := &DeployResponses{
		mutex: sync.Mutex{},
		data:  []*DeployResponse{},
	}
	go func() {
		for {
			select {
			case <-stopCh:
				close(pendingDeploys)
				return
			case r := <-pendingDeploys:
				slog := utils.LogFromContext(r.Context)
				slog.Infof("Processing request: %s", r.Meta)
				errs := []error{}
				for _, bigip := range bigips {
					bc := &f5_bigip.BIGIPContext{BIGIP: *bigip, Context: r.Context}
					if err := HandleRequest(bc, r); err != nil {
						// report status
						slog.Errorf(err.Error())
						errs = append(errs, err)
					}
				}

				resp := DeployResponse{DeployRequest: r, Status: utils.MergeErrors(errs)}
				doneDeploys.Append(&resp)
			}
		}
	}()
	return pendingDeploys, doneDeploys
}

func (dr *DeployResponses) Append(r *DeployResponse) {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	dr.data = append(dr.data, r)
}

func (dr *DeployResponses) Shift() *DeployResponse {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	if len(dr.data) == 0 {
		return nil
	} else {
		f := dr.data[0]
		dr.data = dr.data[1:]
		return f
	}
}

func (dr *DeployResponses) Empty() bool {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	return len(dr.data) == 0
}
