package main

import (
	"context"
	"os"
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

	// create deployer program with given bigip list,
	// returns a chan for accepting DeployRequest and a list for accepting DeployResponse
	reqChan, respList := deployer.Deployer(stopCh, []*f5_bigip.BIGIP{bigip})

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

	// check myvirtual is created.
	for pending := true; pending; pending = respList.Empty() {
		slog.Debugf("waiting for response")
		<-time.After(100 * time.Millisecond)
	}

	if resp := respList.Shift(); resp.Status != nil {
		slog.Errorf("failed to do deployment: %s", resp.Status.Error())
		os.Exit(1)
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
	for pending := true; pending; pending = respList.Empty() {
		slog.Debugf("waiting for response")
		<-time.After(100 * time.Millisecond)
	}
	if resp := respList.Shift(); resp.Status != nil {
		slog.Errorf("failed to do deletion: %s", resp.Status.Error())
		os.Exit(1)
	}
}
