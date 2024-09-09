package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const MetricsPrefix = "vault_kubernetes_kms"

var metricsPrefix = func(s string) string {
	return MetricsPrefix + "_" + s
}

func RegisterPrometheusMetrics() *prometheus.Registry {
	promReg := prometheus.NewRegistry()

	promReg.MustRegister(
		// note: The process collector only collects metrics on Linux OS.
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			Namespace: MetricsPrefix,
		}),
		EncryptionErrorsTotal,
		DecryptionErrorsTotal,
		EncryptionOperationDurationSeconds,
		DecryptionOperationDurationSeconds,
		VaultRequestErrorsTotal,
		VaultRequestDuration,
		VaultTokenRenewalMetricTotal,
		VaultTokenExpirySeconds,
	)

	return promReg
}

var (
	EncryptionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricsPrefix("encryption_errors_total"),
			Help: "total number of errors during encryption operations",
		},
	)

	DecryptionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricsPrefix("decryption_errors_total"),
			Help: "total number of errors during decryption operations",
		},
	)

	EncryptionOperationDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: metricsPrefix("encryption_operation_duration_seconds"),
			Help: "duration of encryption operations",
		},
	)

	DecryptionOperationDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: metricsPrefix("decryption_operation_duration_seconds"),
			Help: "duration of decryption operations",
		},
	)

	VaultRequestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metricsPrefix("vault_requests_errors_total"),
			Help: "total number of errors during API requests sent to vault",
		},
		[]string{"response_code", "path"},
	)

	VaultRequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    metricsPrefix("vault_request_duration_seconds"),
			Help:    "duration of API requests sent to vault",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
	)

	VaultTokenRenewalMetricTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricsPrefix("token_renewals_total"),
			Help: "total number of token renewals",
		},
	)

	VaultTokenExpirySeconds = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: metricsPrefix("token_expiry_seconds"),
			Help: "time remaining until the current token expires",
		},
	)
)
