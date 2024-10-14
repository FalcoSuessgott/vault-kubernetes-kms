package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const MetricsPrefix = "vault_kubernetes_kms"

// nolint: mnd
var defaultBuckets = prometheus.ExponentialBuckets(0.001, 2, 11)

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
		VaultTokenRenewalTotal,
		VaultTokenExpirySeconds,
	)

	return promReg
}

var (
	EncryptionOperationDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    metricsPrefix("encryption_operation_duration_seconds"),
			Help:    "duration of encryption operations",
			Buckets: defaultBuckets,
		},
	)

	DecryptionOperationDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    metricsPrefix("decryption_operation_duration_seconds"),
			Help:    "duration of decryption operations",
			Buckets: defaultBuckets,
		},
	)

	EncryptionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricsPrefix("encryption_operation_errors_total"),
			Help: "total number of errors during encryption operations",
		},
	)

	DecryptionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricsPrefix("decryption_operation_errors_total"),
			Help: "total number of errors during decryption operations",
		},
	)

	VaultTokenRenewalTotal = prometheus.NewCounter(
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
