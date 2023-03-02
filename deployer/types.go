package deployer

import (
	"context"
)

type DeployRequest struct {
	Meta      string
	From      *map[string]interface{}
	To        *map[string]interface{}
	Partition string
	Context   context.Context
}

type CtxKeyType string
