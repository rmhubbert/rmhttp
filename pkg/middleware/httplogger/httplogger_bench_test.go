package httplogger

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// benchIPSink keeps resolver results observable to the compiler without
// adding allocations inside the timed loop.
var benchIPSink string

// newBenchRequest builds a request with a controlled RemoteAddr and
// X-Forwarded-For / X-Real-Ip headers. Requests are built OUTSIDE the timed
// loops so only the resolver call itself is measured.
func newBenchRequest(remoteAddr string, xffLines []string, realIP string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = remoteAddr
	for _, line := range xffLines {
		req.Header.Add("X-Forwarded-For", line)
	}
	if realIP != "" {
		req.Header.Set("X-Real-Ip", realIP)
	}
	return req
}

// Benchmark_realIp measures the per-request client IP resolution hot path.
// The function name and sub-benchmark names are the benchstat contract and
// are intentionally kept byte-identical to the Phase-A baseline capture of
// the original realIp function.
func Benchmark_realIp(b *testing.B) {
	b.ReportAllocs()

	cases := []struct {
		name       string
		remoteAddr string
		xffLines   []string
		realIP     string
	}{
		{"NoHeaders", "203.0.113.7:52000", nil, ""},
		{"XFF_Single", "10.0.0.1:80", []string{"203.0.113.7"}, ""},
		{
			"XFF_Multi_4",
			"10.0.0.1:80",
			[]string{"203.0.113.7, 198.51.100.9, 192.0.2.24, 10.0.0.2"},
			"",
		},
		{"XRealIP", "10.0.0.1:80", nil, "203.0.113.7"},
		{"RemoteAddr_IPv6", "[2001:db8::1]:8080", nil, ""},
		{"XFF_Garbage", "10.0.0.1:80", []string{"not-an-ip, also-not-an-ip"}, ""},
		{"XFF_DupHeaderLines", "10.0.0.1:80", []string{"203.0.113.7", "198.51.100.9"}, ""},
	}

	for _, tc := range cases {
		req := newBenchRequest(tc.remoteAddr, tc.xffLines, tc.realIP)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				benchIPSink = clientIP(req)
			}
		})
	}
}

// Benchmark_realIp_Parallel proves the resolver read path is contention-free
// under concurrent use (pure function, no global logger access).
func Benchmark_realIp_Parallel(b *testing.B) {
	b.ReportAllocs()

	req := newBenchRequest("203.0.113.7:52000", nil, "")
	b.RunParallel(func(pb *testing.PB) {
		var total int
		for pb.Next() {
			total += len(clientIP(req))
		}
		if total >= 0 {
			benchIPSink = req.RemoteAddr // single write per goroutine, keeps call live
		}
	})
}
