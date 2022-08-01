package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	pconfig "github.com/prometheus/common/config"
	"github.com/yuanyp8/http_exporter/conf"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"strings"
)

// AdjustTarget 校验target格式
func AdjustTarget(target string) string {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return fmt.Sprintf("http://%s", target)
	}
	return target
}

// 解析url
func urlParse(src string) (dest *url.URL, host, port string, err error) {
	dest, err = url.Parse(src)
	if err != nil {
		return nil, "", "", err
	}
	return dest, dest.Host, dest.Port(), err
}

func ProbeHTTP(ctx context.Context, target string, module conf.Module, registry *prometheus.Registry) (success bool) {

	var (
		durationGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "probe_http_duration_seconds",
			Help: "Duration of http request by phase, summed over all redirects",
		}, []string{"phase"})

		contentLengthGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_content_length",
			Help: "Length of http content response",
		})

		bodyUncompressedLengthGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_uncompressed_body_length",
			Help: "Length of uncompressed response body",
		})

		redirectsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_redirects",
			Help: "The number of redirects",
		})

		isSSLGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_ssl",
			Help: "Indicates if SSL was used for the final redirect",
		})

		statusCodeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_status_code",
			Help: "Response HTTP status code",
		})

		probeSSLEarliestCertExpiryGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_ssl_earliest_cert_expiry",
			Help: "Returns earliest SSL cert expiry in unixtime",
		})

		probeSSLLastChainExpiryTimestampSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_ssl_last_chain_expiry_timestamp_seconds",
			Help: "Returns last SSL chain expiry in timestamp seconds",
		})

		probeSSLLastInformation = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "probe_ssl_last_chain_info",
				Help: "Contains SSL leaf certificate information",
			},
			[]string{"fingerprint_sha256"},
		)

		probeTLSVersion = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "probe_tls_version_info",
				Help: "Contains the TLS version used",
			},
			[]string{"version"},
		)

		probeHTTPVersionGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_version",
			Help: "Returns the version of HTTP of the probe response",
		})

		probeFailedDueToRegex = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_failed_due_to_regex",
			Help: "Indicates if probe failed due to regex",
		})

		probeHTTPLastModified = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_http_last_modified_timestamp_seconds",
			Help: "Returns the Last-Modified HTTP response header in unixtime",
		})
	)
	registry.MustRegister(
		durationGaugeVec,
		contentLengthGauge,
		bodyUncompressedLengthGauge,
		redirectsGauge,
		isSSLGauge,
		statusCodeGauge,
		probeHTTPVersionGauge,
		probeFailedDueToRegex,
		probeSSLEarliestCertExpiryGauge,
		probeHTTPLastModified,
		probeTLSVersion,
		probeSSLLastInformation,
		probeSSLLastChainExpiryTimestampSeconds
	)

	var redirects int

	httpConfig := module.HTTP
	httpClientConfig := module.HTTP.HTTPClientConfig

	// 自动加上http头
	target = AdjustTarget(target)

	targetUrl, targetHost, targetPort, err := urlParse(target)
	if err != nil {
		l.Error("Could not parse target URL", zap.Error(err))
		return
	}

	// 在没有proxy的情况下进行域名解析
	ip, err := httpConfig.LookUpWithoutProxy(ctx, target, durationGaugeVec)
	if err != nil {
		l.Error("Error resolving address", zap.Error(err))
		return false
	}

	// 大写，替代strings.Upper
	caser := cases.Title(language.Und)

	if len(httpClientConfig.TLSConfig.ServerName) == 0 {
		// 如果tls_config 中没有配置server_name， 就将server_name设置成hostname
		httpClientConfig.TLSConfig.ServerName = targetHost
	}

	// 记录重定向的地址
	// 如果我们在header里配置了重定向的地址，应该用这个地址替代host，目的是为了防止当host为ip时造成的TLS hand shake 失败
	for name, value := range httpConfig.Headers {
		if caser.String(name) == "Host" {
			httpClientConfig.TLSConfig.ServerName = value
		}
	}

	// 基于prometheus的common config生成一个http client，主要作用是配置好了认证服务， e.g. basic auth
	client, err := pconfig.NewClientFromConfig(httpClientConfig, "http_probe", pconfig.WithKeepAlivesDisabled())
	if err != nil {
		l.Error("Error generating HTTP client", zap.Error(err))
		return false
	}

	// host置为空，开始准备NoServerName的情况
	httpClientConfig.TLSConfig.ServerName = ""

	noServerName, err := pconfig.NewRoundTripperFromConfig(httpClientConfig, "http_probe", pconfig.WithKeepAlivesDisabled())
	if err != nil {
		l.Error("Error generating HTTP client without ServerName", zap.Error(err))
		return false
	}

	// 设置http client的cookie
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		l.Error("Error generating cookiejar", zap.Error(err))
		return false
	}
	client.Jar = jar


	tt := newTransport(client.Transport, noServerName)
	client.Transport = tt

	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		l.Info("Received redirect", zap.String("location", r.Response.Header.Get("Location")))
		redirects = len(via)
		if redirects > 10 || !httpConfig.HTTPClientConfig.FollowRedirects {
			l.Info("Not following redirect")
			return errors.New("don't follow redirects")
		}
		return nil
	}

	if httpConfig.Method == "" {
		httpConfig.Method = "GET"
	}

	origHost := targetUrl.Host
	if ip != nil {
		if targetPort == "" {
			if strings.Contains(ip.String(), ":") {
				targetUrl.Host = "[" + ip.String() + "]"
			} else {
				targetUrl.Host = ip.String()
			}
		} else {
			targetUrl.Host = net.JoinHostPort(ip.String(), targetPort)
		}
	}
	var body io.Reader
	var respBodyBytes int64

	// If a body is configured, add it to the request.
	if httpConfig.Body != "" {
		body = strings.NewReader(httpConfig.Body)
	}

	request, err := http.NewRequest(httpConfig.Method, targetUrl.String(), body)
	if err != nil {
		l.Error("Error creating request", zap.Error(err))
		return
	}

	request.Host = origHost
	request = request.WithContext(ctx)

	// 将配置文件中声明的Header 添加到request中
	for key, value := range httpConfig.Headers {
		if caser.String(key) == "Host" {
			request.Host = value
			continue
		}

		request.Header.Set(key, value)
	}

	// 设置默认User-Agent
	_, hasUserAgent := request.Header["User-Agent"]
	if !hasUserAgent {
		request.Header.Set("User-Agent", userAgentDefaultHeader)
	}

	trace := &httptrace.ClientTrace{
		DNSStart:             tt.DNSStart,
		DNSDone:              tt.DNSDone,
		ConnectStart:         tt.ConnectStart,
		ConnectDone:          tt.ConnectDone,
		GotConn:              tt.GotConn,
		GotFirstResponseByte: tt.GotFirstResponseByte,
		TLSHandshakeStart:    tt.TLSHandshakeStart,
		TLSHandshakeDone:     tt.TLSHandshakeDone,
	}

	request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))

	// 批量增加metrics label
	for _, lv := range []string{"connect", "tls", "processing", "transfer"} {
		durationGaugeVec.WithLabelValues(lv)
	}

	resp, err := client.Do(request)

	if resp == nil {
		resp = &http.Response{}
		if err != nil {
			l.Error("Error for HTTP request", zap.Error(err))
		}
	} else {
		requestErrored := (err != nil)
		l.Info("Received HTTP response", zap.Int("status_code", resp.StatusCode))
		if len(httpConfig.ValidStatusCode) != 0 {
			for _, code := range httpConfig.ValidStatusCode {
				if resp.StatusCode == code {
					success = true
					break
				}
			}
			if !success {
				l.Info("Invalid HTTP response status code",
					zap.Int("status_code", resp.StatusCode),
					zap.String("valid_status_codes", fmt.Sprintf("%v", httpConfig.ValidStatusCode)))
			}
		} else if 20 <= resp.StatusCode && resp.StatusCode < 300 {
			success = true
		} else {
			l.Info("Invalid HTTP response status code, wanted 2xx", zap.Int("status_code", resp.StatusCode))
		}

		if success && (len(httpConfig.FailIfHeaderMatchesRegexp) > 0 || len(httpConfig.FailIfHeaderNotMatchesRegexp) > 0) {

		}

	}



	return
}
