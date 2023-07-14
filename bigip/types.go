package f5_bigip

import (
	"context"
	"net/http"
)

type RestRequest struct {
	ResName   string
	Partition string
	Subfolder string
	Kind      string

	Method     string
	ResUri     string
	Headers    map[string]interface{}
	Body       interface{}
	WithTrans  bool
	ScheduleIt string
}

type BIGIP struct {
	Version       string
	URL           string
	Authorization string
	client        *http.Client
}

type BIGIPContext struct {
	BIGIP
	context.Context
}
type BIGIPVersion struct {
	Build   string
	Date    string
	Edition string
	Product string
	Title   string
	Version string
}
