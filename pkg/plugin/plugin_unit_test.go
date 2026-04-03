package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	v1beta1 "k8s.io/kms/apis/v1beta1"
	v2 "k8s.io/kms/apis/v2"
)

type requestContextKey struct{}

type fakePlugin struct {
	encryptResponse []byte
	decryptResponse []byte
	keyVersion      string
	encryptErr      error
	decryptErr      error
	keyVersionErr   error
	encryptValue    any
	decryptValue    any
	keyVersionValue any
}

func (f *fakePlugin) Encrypt(ctx context.Context, data []byte) ([]byte, string, error) {
	f.encryptValue = ctx.Value(requestContextKey{})

	return f.encryptResponse, f.keyVersion, f.encryptErr
}

func (f *fakePlugin) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	f.decryptValue = ctx.Value(requestContextKey{})

	return f.decryptResponse, f.decryptErr
}

func (f *fakePlugin) GetKeyVersion(ctx context.Context) (string, error) {
	f.keyVersionValue = ctx.Value(requestContextKey{})

	return f.keyVersion, f.keyVersionErr
}

func resetPluginMetrics() {
	metrics.EncryptionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "vault_kubernetes_kms_encryption_operation_errors_total",
			Help: "total number of errors during encryption operations",
		},
	)
	metrics.DecryptionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "vault_kubernetes_kms_decryption_operation_errors_total",
			Help: "total number of errors during decryption operations",
		},
	)
	metrics.EncryptionOperationDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "vault_kubernetes_kms_encryption_operation_duration_seconds",
			Help: "duration of encryption operations",
		},
	)
	metrics.DecryptionOperationDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "vault_kubernetes_kms_decryption_operation_duration_seconds",
			Help: "duration of decryption operations",
		},
	)
}

func histogramSampleCount(t *testing.T, collector prometheus.Collector) uint64 {
	t.Helper()

	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Len(t, families[0].GetMetric(), 1)
	metric := families[0].GetMetric()[0]

	return metric.GetHistogram().GetSampleCount()
}

func counterValue(t *testing.T, collector prometheus.Collector) float64 {
	t.Helper()

	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Len(t, families[0].GetMetric(), 1)
	metric := families[0].GetMetric()[0]

	return metric.GetCounter().GetValue()
}

func TestKMSv1UsesPluginInterface(t *testing.T) {
	kms := NewPluginV1(&fakePlugin{
		encryptResponse: []byte("cipher"),
		decryptResponse: []byte("plain"),
		keyVersion:      "1",
	})

	//nolint:staticcheck // KMS v1 coverage is intentional here.
	enc, err := kms.Encrypt(context.Background(), &v1beta1.EncryptRequest{Plain: []byte("plain")})
	require.NoError(t, err)
	//nolint:staticcheck // KMS v1 coverage is intentional here.
	require.Equal(t, []byte("cipher"), enc.GetCipher())

	//nolint:staticcheck // KMS v1 coverage is intentional here.
	dec, err := kms.Decrypt(context.Background(), &v1beta1.DecryptRequest{Cipher: []byte("cipher")})
	require.NoError(t, err)
	//nolint:staticcheck // KMS v1 coverage is intentional here.
	require.Equal(t, []byte("plain"), dec.GetPlain())
}

func TestKMSv2StatusReflectsHealthFailures(t *testing.T) {
	resetPluginMetrics()

	kms := NewPluginV2(&fakePlugin{
		encryptResponse: []byte("cipher"),
		decryptResponse: []byte("wrong"),
		keyVersion:      "7",
	})

	resp, err := kms.Status(context.Background(), &v2.StatusRequest{})
	require.NoError(t, err)
	require.Equal(t, "v2", resp.GetVersion())
	require.Equal(t, "err", resp.GetHealthz())
	require.Equal(t, "7", resp.GetKeyId())
	require.Zero(t, counterValue(t, metrics.EncryptionErrorsTotal))
	require.Zero(t, counterValue(t, metrics.DecryptionErrorsTotal))
	require.Zero(t, histogramSampleCount(t, metrics.EncryptionOperationDurationSeconds))
	require.Zero(t, histogramSampleCount(t, metrics.DecryptionOperationDurationSeconds))
}

func TestKMSv2StatusReturnsKeyVersionErrors(t *testing.T) {
	kms := NewPluginV2(&fakePlugin{
		keyVersionErr: errors.New("vault unavailable"),
	})

	_, err := kms.Status(context.Background(), &v2.StatusRequest{})
	require.EqualError(t, err, "vault unavailable")
}

func TestKMSv2StatusUsesRequestContextForHealth(t *testing.T) {
	resetPluginMetrics()

	fake := &fakePlugin{
		encryptResponse: []byte("cipher"),
		decryptResponse: []byte("health"),
		keyVersion:      "9",
	}
	kms := NewPluginV2(fake)

	ctx := context.WithValue(context.Background(), requestContextKey{}, "status")

	resp, err := kms.Status(ctx, &v2.StatusRequest{})
	require.NoError(t, err)
	require.Equal(t, "ok", resp.GetHealthz())
	require.Equal(t, "status", fake.keyVersionValue)
	require.Equal(t, "status", fake.encryptValue)
	require.Equal(t, "status", fake.decryptValue)
	require.Zero(t, histogramSampleCount(t, metrics.EncryptionOperationDurationSeconds))
	require.Zero(t, histogramSampleCount(t, metrics.DecryptionOperationDurationSeconds))
}

func TestKMSv2EncryptRecordsMetricsOnlyForNormalTraffic(t *testing.T) {
	resetPluginMetrics()

	kms := NewPluginV2(&fakePlugin{
		encryptResponse: []byte("cipher"),
		keyVersion:      "1",
	})

	_, err := kms.Encrypt(context.Background(), &v2.EncryptRequest{
		Plaintext: []byte("plain"),
		Uid:       "uid",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), histogramSampleCount(t, metrics.EncryptionOperationDurationSeconds))
	require.Zero(t, counterValue(t, metrics.EncryptionErrorsTotal))
}

func TestKMSv2DecryptRecordsMetricsOnlyForNormalTraffic(t *testing.T) {
	resetPluginMetrics()

	kms := NewPluginV2(&fakePlugin{
		decryptResponse: []byte("plain"),
	})

	_, err := kms.Decrypt(context.Background(), &v2.DecryptRequest{
		Ciphertext: []byte("cipher"),
		Uid:        "uid",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), histogramSampleCount(t, metrics.DecryptionOperationDurationSeconds))
	require.Zero(t, counterValue(t, metrics.DecryptionErrorsTotal))
}
