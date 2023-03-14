package deployer

import (
	"context"
	"sync"
)

type DeployRequest struct {
	Meta      string
	From      *map[string]interface{}
	To        *map[string]interface{}
	Partition string
	Context   context.Context
}

type DeployResponse struct {
	DeployRequest
	Status error
}

type DeployResponses struct {
	data  []*DeployResponse
	mutex sync.Mutex
}

type CtxKeyType string
