package f5_bigip

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	utils "github.com/f5devcentral/f5-bigip-rest-go/utils"
)

func Test_assertBigipResp20X(t *testing.T) {
	type Case struct {
		code     int
		resp     []byte
		expected bool
	}

	cases := []Case{
		{200, []byte(""), false},
		{401, []byte(`{"code":401,"message":"Authorization failed: no user authentication header or token detected. Uri:http://localhost:8100/mgmt/tm/ltm/virtual Referrer:10.145.74.44 Sender:10.145.74.44","referer":"10.145.74.44","restOperationId":7945916,"kind":":resterrorresponse"}`), true},
		{404, []byte(`{"code":404,"message":"URI path /mgmt/tm/ltm/pool/?Common?my-pool not registered.  Please verify URI is supported and wait for /available suffix to be responsive.","restOperationId":41,"kind":":resterrorresponse"}`), true},
		{404, []byte(`{"code":404,"message":"Public URI path not registered: /tm/ltm/pool/?Common?my-pool","referer":"10.250.64.100","restOperationId":39168,"kind":":resterrorresponse"`), true},
		{404, []byte(`{"code":404,"message":"01020036:3: The requested Pool (/Common/my-pool) was not found.","errorStack":[],"apiError":3}`), false},
		{500, []byte(`{"code":500,"message":"The connection to mcpd has been lost, try again.","errorStack":[],"apiError":32768001}`), true},
		{503, []byte("long html response..: Configuration Utility restarting..."), true},
	}

	for _, c := range cases {
		err := assertBigipResp20X(c.code, c.resp)
		if utils.NeedRetry(err) != c.expected {
			t.Fail()
		}
	}
}

func Test_sweepCmds(t *testing.T) {
	type args struct {
		dels      map[string][]RestRequest
		crts      map[string][]RestRequest
		existings *map[string]map[string]interface{}
	}
	tests := []struct {
		name  string
		args  args
		creat []RestRequest
		delet []RestRequest
		updat []RestRequest
	}{
		// empty input
		{
			name: "empty input",
			args: args{
				map[string][]RestRequest{},
				map[string][]RestRequest{},
				&map[string]map[string]interface{}{},
			},
			creat: []RestRequest{},
			delet: []RestRequest{},
			updat: []RestRequest{},
		},
		// pure create
		{
			name: "pure create",
			args: args{
				map[string][]RestRequest{},
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				&map[string]map[string]interface{}{},
			},
			creat: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node1",
					Method:    "POST",
					Kind:      "ltm/node",
				},
			},
			delet: []RestRequest{},
			updat: []RestRequest{},
		},
		// pure delete
		{
			name: "pure delete",
			args: args{
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				map[string][]RestRequest{},
				&map[string]map[string]interface{}{
					"ltm/node": {
						utils.Keyname("p1", "", "node1"): map[string]interface{}{},
					},
				},
			},
			creat: []RestRequest{},
			delet: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node1",
					Method:    "DELETE",
					Kind:      "ltm/node",
				},
			},
			updat: []RestRequest{},
		},
		// pure update
		{
			name: "pure update",
			args: args{
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				&map[string]map[string]interface{}{
					"ltm/node": {
						utils.Keyname("p1", "", "node1"): map[string]interface{}{},
					},
				},
			},
			creat: []RestRequest{},
			delet: []RestRequest{},
			updat: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node1",
					Method:    "PATCH",
					Kind:      "ltm/node",
				},
			},
		},
		// mix create delete update
		{
			name: "mix create delete update",
			args: args{
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node2",
							Kind:      "ltm/node",
						},
					},
				},
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node3",
							Kind:      "ltm/node",
						},
					},
				},
				&map[string]map[string]interface{}{
					"ltm/node": {
						utils.Keyname("p1", "", "node1"): map[string]interface{}{},
						utils.Keyname("p1", "", "node2"): map[string]interface{}{},
					},
				},
			},
			creat: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node3",
					Kind:      "ltm/node",
					Method:    "POST",
				},
			},
			delet: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node2",
					Kind:      "ltm/node",
					Method:    "DELETE",
				},
			},
			updat: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node1",
					Method:    "PATCH",
					Kind:      "ltm/node",
				},
			},
		},
		// create to update
		{
			name: "create to update",
			args: args{
				map[string][]RestRequest{},
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				&map[string]map[string]interface{}{
					"ltm/node": {
						utils.Keyname("p1", "", "node1"): map[string]interface{}{},
					},
				},
			},
			creat: []RestRequest{},
			delet: []RestRequest{},
			updat: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node1",
					Method:    "PATCH",
					Kind:      "ltm/node",
				},
			},
		},
		// update to create
		{
			name: "update to create",
			args: args{
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				&map[string]map[string]interface{}{},
			},
			creat: []RestRequest{
				{
					Partition: "p1",
					Subfolder: "",
					ResName:   "node1",
					Method:    "POST",
					Kind:      "ltm/node",
				},
			},
			delet: []RestRequest{},
			updat: []RestRequest{},
		},
		// delete to nope
		{
			name: "delete to nope",
			args: args{
				map[string][]RestRequest{
					"ltm/node": {
						{
							Partition: "p1",
							Subfolder: "",
							ResName:   "node1",
							Kind:      "ltm/node",
						},
					},
				},
				map[string][]RestRequest{},
				&map[string]map[string]interface{}{},
			},
			creat: []RestRequest{},
			delet: []RestRequest{},
			updat: []RestRequest{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, d, u := sweepCmds(tt.args.dels, tt.args.crts, tt.args.existings)
			if !reflect.DeepEqual(c, tt.creat) {
				t.Errorf("sweepCmds() c = %v, want %v", c, tt.creat)
			}
			if !reflect.DeepEqual(d, tt.delet) {
				t.Errorf("sweepCmds() d = %v, want %v", d, tt.delet)
			}
			if !reflect.DeepEqual(u, tt.updat) {
				t.Errorf("sweepCmds() u = %v, want %v", u, tt.updat)
			}
		})
	}
}

func Test_layoutCmds(t *testing.T) {
	type args struct {
		c []RestRequest
		d []RestRequest
		u []RestRequest
	}

	folder := RestRequest{
		ResName:   "f1",
		Partition: "p1",
		Subfolder: "",
		Kind:      "sys/folder",
	}
	virtual := RestRequest{
		ResName:   "v1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/virtual",
	}
	pool := RestRequest{
		ResName:   "p1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/pool",
	}

	monitor := RestRequest{
		ResName:   "m1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/monitor/http",
	}

	node := RestRequest{
		ResName:   "n1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/node",
	}

	arp := RestRequest{
		ResName:   "a1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "net/arp",
	}

	snatpool := RestRequest{
		ResName:   "sp1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/snatpool",
	}

	profile := RestRequest{
		ResName:   "p1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/profile/http",
	}

	rule := RestRequest{
		ResName:   "r1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/rule",
	}

	virtualAddress := RestRequest{
		ResName:   "va1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "ltm/virtual-address",
	}

	fdb := RestRequest{
		ResName:   "f1",
		Partition: "p1",
		Subfolder: "f1",
		Kind:      "net/fdb/tunnel",
	}

	tests := []struct {
		name string
		args args
		want []RestRequest
	}{
		{
			name: "empty",
			args: args{
				c: []RestRequest{},
				d: []RestRequest{},
				u: []RestRequest{},
			},
			want: []RestRequest{},
		},
		{
			name: "only c",
			args: args{
				c: []RestRequest{
					virtual, pool, node, arp, monitor, folder,
					fdb, virtualAddress, snatpool, profile, rule,
				},
				d: []RestRequest{},
				u: []RestRequest{},
			},
			want: []RestRequest{
				folder, monitor, node, pool, snatpool,
				profile, rule, virtualAddress, virtual, arp, fdb,
			},
		},
		{
			name: "c and u",
			args: args{
				c: []RestRequest{
					virtual, pool, node, arp, monitor, folder,
				},
				d: []RestRequest{},
				u: []RestRequest{
					fdb, virtualAddress, snatpool, profile, rule,
				},
			},
			want: []RestRequest{
				folder, monitor, node, pool, snatpool, profile,
				rule, virtualAddress, virtual, arp, fdb,
			},
		},
		{
			name: "c and d",
			args: args{
				c: []RestRequest{
					virtual, pool, node, arp, monitor, folder,
				},
				d: []RestRequest{
					fdb, virtualAddress, snatpool, profile, rule,
				},
				u: []RestRequest{},
			},
			want: []RestRequest{
				folder, monitor, node, pool, virtual, arp,
				fdb, virtualAddress, rule, profile, snatpool,
			},
		},
		{
			name: "u and d",
			args: args{
				c: []RestRequest{},
				d: []RestRequest{
					fdb, virtualAddress, snatpool, profile, rule,
				},
				u: []RestRequest{
					virtual, pool, node, arp, monitor, folder,
				},
			},
			want: []RestRequest{
				folder, monitor, node, pool, virtual, arp,
				fdb, virtualAddress, rule, profile, snatpool,
			},
		},
		{
			name: "c u d",
			args: args{
				c: []RestRequest{
					arp, monitor, virtualAddress, snatpool,
				},
				d: []RestRequest{
					fdb, profile, rule,
				},
				u: []RestRequest{
					virtual, pool, node, folder,
				},
			},
			want: []RestRequest{
				folder, monitor, node, pool, snatpool,
				virtualAddress, virtual, arp,
				fdb, rule, profile,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := layoutCmds(tt.args.c, tt.args.d, tt.args.u); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("layoutCmds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_refname(t *testing.T) {
	type args struct {
		partition string
		subfolder string
		name      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				partition: "partition",
				subfolder: "subfolder",
				name:      "pool",
			},
			want: "~partition~subfolder~pool",
		},
		{
			name: "empty subfolder",
			args: args{
				partition: "partition",
				subfolder: "",
				name:      "pool",
			},
			want: "~partition~pool",
		},
		{
			name: "route-domained resource",
			args: args{
				partition: "partition",
				subfolder: "",
				name:      "pool%23",
			},
			want: "~partition~pool%2523",
		},

		{
			name: "pathed resource",
			args: args{
				partition: "partition",
				subfolder: "",
				name:      "pool/members",
			},
			want: "~partition~pool/members",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.Refname(tt.args.partition, tt.args.subfolder, tt.args.name); got != tt.want {
				t.Errorf("refname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKindIsSupported(t *testing.T) {
	tests := []string{
		// TODO: Add test cases.
		"sys/folder true",
		"net/routing false",
		"net/routing/bgp true",
		"sys/file/ssl-cert true",
		"ltm/monitor/http true",
		"ltm/data-group/internal true",
		"ltm/virtualaaa false",
		"net/fdb/tunnel/~Common~fl-tunnel/records true",
		"net/routing/bgp/~Common~k8s-bgp/neighbor true",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			kind := strings.Split(tt, " ")[0]
			bl := strings.Split(tt, " ")[1]
			want, _ := strconv.ParseBool(bl)
			if got := KindIsSupported(kind); got != want {
				t.Errorf("KindIsSupported(%s) = %v, want %v", kind, got, want)
			}
		})
	}
}
