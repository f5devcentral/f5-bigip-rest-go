package main

import (
	"context"
	"time"

	f5_bigip "github.com/f5devcentral/f5-bigip-rest-go/bigip"
	// import deployer module
	"github.com/f5devcentral/f5-bigip-rest-go/deployer"
	"github.com/f5devcentral/f5-bigip-rest-go/utils"
)

func main() {
	// bigip := f5_bigip.New("https://1.2.3.4", "admin", "password")
	bigip := f5_bigip.New("https://10.250.2.218", "admin", "P@ssw0rd123")

	stopCh := make(chan struct{})
	defer close(stopCh)

	// create deployer program with given bigip list,
	// returns a chan for accepting DeployRequest and a list for accepting DeployResponse
	reqQueue, respList := deployer.Deployer(stopCh, []*f5_bigip.BIGIP{bigip})

	slog := utils.NewLog().WithLevel(utils.LogLevel_Type_DEBUG)
	ctx := context.WithValue(context.TODO(), utils.CtxKey_Logger, slog)
	go func() {
		for {
			// check the resources are created/deleted.
			resp := respList.Get().(deployer.DeployResponse)
			if resp.Status != nil {
				slog.Errorf("failed to do deployment: %s", resp.Status.Error())
			} else {
				slog.Infof("done of deploying %s", resp.DeployRequest.Partition)
			}
		}
	}()

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
	reqQueue.Add(deployer.DeployRequest{
		Meta:      "test deployment with deployer",
		From:      nil,
		To:        &ncfgs,
		Partition: partition,
		Context:   lctx1,
	})

	// lctx2 tells the deployer to delete partition after resource deletion.
	lctx2 := context.WithValue(ctx, deployer.CtxKey_DeletePartition, "yes")

	// post the deletion request to channel.
	reqQueue.Add(deployer.DeployRequest{
		Meta:      "test deletion with deployer",
		From:      &ncfgs,
		To:        nil,
		Partition: partition,
		Context:   lctx2,
	})

	<-time.After(20 * time.Second)
}
