package f5_bigip

import (
	"net/http"
	"reflect"
	"testing"
)

func TestBIGIP_GenRestRequests(t *testing.T) {
	type fields struct {
		Version       string
		URL           string
		Authorization string
		client        *http.Client
	}
	type args struct {
		partition string
		ocfg      *map[string]interface{}
		ncfg      *map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *[]RestRequest
		wantErr bool
	}{
		// TODO: Add test cases.

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bip := &BIGIP{
				Version:       tt.fields.Version,
				URL:           tt.fields.URL,
				Authorization: tt.fields.Authorization,
				client:        tt.fields.client,
			}
			got, err := bip.GenRestRequests(tt.args.partition, tt.args.ocfg, tt.args.ncfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("BIGIP.GenRestRequests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BIGIP.GenRestRequests() = %v, want %v", got, tt.want)
			}
		})
	}
}
