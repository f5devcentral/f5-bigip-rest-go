package f5_bigip

import (
	"testing"

	"gitee.com/zongzw/f5-bigip-rest/utils"
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
