package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func gatherVaultRequestMetric(t *testing.T, method, path, status string) *dto.Metric {
	t.Helper()

	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(metrics.VaultRequestsDurationSeconds))
	t.Cleanup(func() {
		require.NoError(t, registry.Unregister(metrics.VaultRequestsDurationSeconds))
	})

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "vault_kubernetes_kms_vault_requests_duration_seconds" {
			continue
		}

		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			if labels["method"] == method && labels["path"] == path && labels["status"] == status {
				return metric
			}
		}
	}

	t.Fatalf("metric not found for %s %s %s", method, path, status)

	return nil
}

func TestRoundTripRecordsStatusCode(t *testing.T) {
	metrics.VaultRequestsDurationSeconds.Reset()

	client := &RoundTripper{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusCreated}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodPost, "https://vault.example/v1/transit/encrypt/kms", nil)
	require.NoError(t, err)

	_, err = client.RoundTrip(req)
	require.NoError(t, err)

	metric := gatherVaultRequestMetric(t, http.MethodPost, "/v1/transit/encrypt/kms", "201")
	require.EqualValues(t, 1, metric.GetHistogram().GetSampleCount())
}

func TestRoundTripRecordsTransportErrors(t *testing.T) {
	metrics.VaultRequestsDurationSeconds.Reset()

	client := &RoundTripper{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("transport failed")
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "https://vault.example/v1/transit/decrypt/kms", nil)
	require.NoError(t, err)

	_, err = client.RoundTrip(req)
	require.EqualError(t, err, "transport failed")

	metric := gatherVaultRequestMetric(t, http.MethodGet, "/v1/transit/decrypt/kms", "error")
	require.EqualValues(t, 1, metric.GetHistogram().GetSampleCount())
}
