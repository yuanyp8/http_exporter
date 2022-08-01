package http

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/yuanyp8/http_exporter/conf"
	"go.uber.org/zap"
	"net"
	"net/url"
	"strings"
)

func ProbeHTTP(ctx context.Context, target string, module conf.Module, registry *prometheus.Registry) (success bool) {
	var redirects int

	// 将方法注册到prometheus
	registied(registry)

	httpConfig := module.HTTP

	// 自动加上http头
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = fmt.Sprintf("http://%s", target)
	}

	targetUrl, err := url.Parse(target)
	if err != nil {
		l.Error("Could not parse target URL", zap.Error(err))
		return
	}

	// parse succeed
	targetHost := targetUrl.Host
	targetPort := targetUrl.Port()

	var ip *net.IPAddr
	if !module.HTTP.SkipResolvePhaseWithProxy || module.HTTP.HTTPClientConfig.ProxyURL.URL == nil {
		var lookUpTime float64
		ip, lookUpTime, err =
	}

}
