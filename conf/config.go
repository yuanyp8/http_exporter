package conf

import (
	"context"
	"github.com/alecthomas/units"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/config"
	"github.com/yuanyp8/http_exporter/utils"
	"net"
	"net/http"
	"regexp"
	"time"
)

var (
	l = utils.Logger.Named("Conf")
)

type Config struct {
	Modules map[string]Module `mapstructure:"modules"`
}

type Module struct {
	Prober  string        `mapstructure:"prober" validate:"required"`
	Timeout time.Duration `mapstructure:"timeout"`
	HTTP    *HTTPProbe    `mapstructure:"http"`
}

func NewDefaultModule() *Module {
	return &Module{
		HTTP: NewDefaultHTTPProbe(),
	}
}

type HTTPProbe struct {
	ValidStatusCode              []int                   `mapstructure:"valid_status_code"`             // Verify response code
	ValidHTTPVersions            []string                `mapstructure:"valid_status_code"`             // Adapt to HTTP1.x/HTTP2
	IPProtocol                   IPProtocol              `mapstructure:"preferred_ip_protocol"`         // Adapt to IPV4/IPV6
	IPProtocolFallback           bool                    `mapstructure:"ip_protocol_fallback"`          // 允许IPV6协议降级
	SkipResolvePhaseWithProxy    bool                    `mapstructure:"skip_resolve_phase_with_proxy"` // 解析域名时不使用代理
	NoFollowRedirects            *bool                   `mapstructure:"no_follow_redirects"`           // 禁止重定向
	FailIfSSL                    bool                    `mapstructure:"fail_if_ssl"`                   // 如果被监控项为HTTPS，则失败
	FailIfNotSSL                 bool                    `mapstructure:"fail_if_not_ssl"`               // 如果被监控项不是HTTPS，则失败
	Method                       string                  `mapstructure:"method"`
	Headers                      map[string]string       `mapstructure:"headers"`                     // Request Headers
	FailIfBodyMatchesRegexp      []Regexp                `mapstructure:"fail_if_body_matches_regexp"` // if Response Headers not include origin strings, return failed  Regexp是对regex.Regexp的封装，包含了源正则字符串
	FailIfBodyNotMatchesRegexp   []Regexp                `mapstructure:"fail_if_body_not_matches_regexp"`
	FailIfHeaderMatchesRegexp    []HeaderMatch           `mapstructure:"fail_if_header_matches"`
	FailIfHeaderNotMatchesRegexp []HeaderMatch           `mapstructure:"fail_if_header_not_matches"`
	Body                         string                  `mapstructure:"body,omitempty"`
	Compression                  string                  `mapstructure:"compression"`        // 指定压缩算法 e.g. gzip
	BodySizeLimit                units.Base2Bytes        `mapstructure:"body_size_limit"`    // units是一个单位转换工作 e.g. 1Mi => 1024*1024
	HTTPClientConfig             config.HTTPClientConfig `mapstructure:"http_client_config"` // prometheus 官方的工具包，包括了BearToken、BasicAuth、TLS、SSL等协议的认证，主要作用是配置http request
}

func NewDefaultHTTPProbe() *HTTPProbe {
	return &HTTPProbe{
		IPProtocol:         IPV4,
		IPProtocolFallback: true,
		Method:             http.MethodGet,
		HTTPClientConfig:   config.DefaultHTTPClientConfig,
	}
}

type Regexp struct {
	*regexp.Regexp
	origin string `mapstructure:"origin"`
}

func NewRegexp(regexExpr string) (*Regexp, error) {
	regex, err := regexp.Compile(regexExpr)
	return &Regexp{regex, regexExpr}, err
}

// MustNewRegexp works like NewRegexp, but panics if the regular expression does not compile.
func MustNewRegexp(regexExpr string) *Regexp {
	re, err := NewRegexp(regexExpr)
	if err != nil {
		panic(err)
	}
	return re
}

type HeaderMatch struct {
	Header       string `mapstructure:"header"`
	Regexp       Regexp `mapstructure:"regexp"`
	AllowMissing bool   `mapstructure:"allow_missing"` // 是否允许不含value
}

func (h HTTPProbe) LookUpWithoutProxy(ctx context.Context, target string, durationGaugeVec *prometheus.GaugeVec) (ip *net.IPAddr, err error) {
	var lookUpTime float64

	if h.SkipResolvePhaseWithProxy || h.HTTPClientConfig.ProxyURL.URL == nil {
		ip, lookUpTime, err = h.ChooseProtocol(ctx, target)
		durationGaugeVec.WithLabelValues("resolve").Add(lookUpTime)
	}
	return
}
