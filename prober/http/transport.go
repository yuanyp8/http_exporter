package http

import (
	"fmt"
	"github.com/prometheus/common/version"
	"github.com/yuanyp8/http_exporter/utils"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

var l = utils.Logger.Named("HTTP")
var userAgentDefaultHeader = fmt.Sprintf("Blackbox Exporter/%s", version.Version)

// NewRequest -> client.Do -> transport.Transport
type transport struct {
	Transport             http.Transport
	NoServerNameTransport http.Transport // 针对target为ip的场景
	firstHost             string
	mu                    sync.Mutex
	traces                []*roundTripTrace
	current               *roundTripTrace
}

func newTransport(rt, noServerName http.Transport) *transport {
	return &transport{
		Transport:             rt,
		NoServerNameTransport: noServerName,
		traces:                []*roundTripTrace{},
	}
}

// 记录一次http监测的生命周期
type roundTripTrace struct {
	tls           bool
	start         time.Time
	dnsDone       time.Time
	connectDone   time.Time
	gotDone       time.Time
	responseStart time.Time
	end           time.Time
	tlsStart      time.Time
	tlsDone       time.Time
}

// RoundTrip 对http client RoundTrip的一层封装
// 实现RoundTripper接口
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	l.Info("Making HTTP request", zap.String("url", req.URL.String()), zap.String("host", req.Host))

	t.current = &roundTripTrace{}
	if req.URL.Scheme == "https" {
		t.current.tls = true
	}
	t.traces = append(t.traces, t.current)

	if t.firstHost == "" {
		t.firstHost = req.URL.Host
	}

	// redirect
	if t.firstHost != req.URL.Host {
		// 发生了重定向
		l.Info("Address does not match first address, not sending TLS ServerName", zap.String("first", t.firstHost), zap.String("address", req.URL.Host))
		// RoundTrip可以理解为自带的连接池管理功能，支持连接重用
		return t.NoServerNameTransport.RoundTrip(req)
	}
	return t.Transport.RoundTrip(req)
}
