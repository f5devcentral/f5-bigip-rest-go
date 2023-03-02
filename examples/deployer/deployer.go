package main

import (
	"context"
	"time"

	f5_bigip "github.com/zongzw/f5-bigip-rest/bigip"
	// import deployer module
	"github.com/zongzw/f5-bigip-rest/deployer"
	"github.com/zongzw/f5-bigip-rest/utils"
)

func main() {
	bigip := f5_bigip.New("https://1.2.3.4", "admin", "password")

	stopCh := make(chan struct{})
	defer close(stopCh)

	// create deployer program with given bigip list, returns a chan for accepting DeployRequest
	reqChan := deployer.Deployer(stopCh, []*f5_bigip.BIGIP{bigip})

	slog := utils.NewLog().WithLevel(utils.LogLevel_Type_DEBUG)
	ctx := context.WithValue(context.TODO(), utils.CtxKey_Logger, slog)

	// deployment test data.
	partition := "mypartition"
	ncfgs := map[string]interface{}{
		"": map[string]interface{}{
			"ltm/virtual/myvirtual": map[string]interface{}{
				"name":        "myvirtual",
				"destination": "197.14.222.12:80",
				"ipProtocol":  "tcp",
				"virtualType": "standard",
			},
		},
	}

	// lctx1 tells the deployer to do the partition creation before resource deployment.
	lctx1 := context.WithValue(ctx, deployer.CtxKey_CreatePartition, "yes")

	// post the deploy request to the channel.
	reqChan <- deployer.DeployRequest{
		Meta:      "test deployment with deployer",
		From:      nil,
		To:        &ncfgs,
		Partition: partition,
		Context:   lctx1,
	}

	// used to check bigip resource creation/deletion.
	bc := f5_bigip.BIGIPContext{BIGIP: *bigip, Context: ctx}

	// check myvirtual is created.
	for {
		slog.Debugf("check virtual existing")
		<-time.After(1 * time.Second)
		if existings, err := bc.GetExistingResources(partition, []string{"ltm/virtual"}); err != nil {
			slog.Errorf("failed to get existing resources: %s", err.Error())
			break
		} else {
			if v, ok := (*existings)["ltm/virtual"]; ok {
				if _, found := v[utils.Keyname(partition, "myvirtual")]; found {
					slog.Infof("resource %s/%s created.", partition, "myvirtual")
					break
				}
			}
		}
	}

	// lctx2 tells the deployer to delete partition after resource deletion.
	lctx2 := context.WithValue(ctx, deployer.CtxKey_DeletePartition, "yes")

	// post the deletion request to channel.
	reqChan <- deployer.DeployRequest{
		Meta:      "test deletion with deployer",
		From:      &ncfgs,
		To:        nil,
		Partition: partition,
		Context:   lctx2,
	}

	// check the resource is deleted.
	for {
		slog.Debugf("check virtual deleted")
		<-time.After(1 * time.Second)
		if existings, err := bc.GetExistingResources(partition, []string{"ltm/virtual"}); err != nil {
			slog.Errorf("failed to get existing resources: %s", err.Error())
			break
		} else {
			if v, ok := (*existings)["ltm/virtual"]; !ok {
				slog.Infof("resource kind ltm/virtual not found.")
				break
			} else {
				if _, found := v[utils.Keyname(partition, "myvirtual")]; !found {
					slog.Infof("resource %s/%s deleted.", partition, "myvirtual")
					break
				}
			}
		}
	}
}
