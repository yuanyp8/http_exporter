package conf

import "github.com/prometheus/client_golang/prometheus"

var (
	configReloadSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "http_exporter",
		Name:      "config_last_reload_successful",
		Help:      "Http exporter config loaded successfully.",
	})

	configReloadSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "http_exporter",
		Name:      "config_last_reload_success_timestamp_seconds",
		Help:      "Timestamp of the last successful configuration reload.",
	})
)

func init() {
	prometheus.MustRegister(configReloadSuccess, configReloadSeconds)
}
