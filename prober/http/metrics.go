package http

import "github.com/prometheus/client_golang/prometheus"

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

func registied(registry *prometheus.Registry) {
	registry.MustRegister(
		durationGaugeVec,
		contentLengthGauge,
		bodyUncompressedLengthGauge,
		redirectsGauge,
		isSSLGauge,
		statusCodeGauge,
		probeHTTPVersionGauge,
		probeFailedDueToRegex,
	)
}
