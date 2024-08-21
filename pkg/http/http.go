package http

import (
	"net/http"
	"strconv"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPClient is a custom HTTP client that measures the duration of requests.
type HTTPClient struct {
	*http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Transport: &TimedRoundTripper{
				Transport: http.DefaultTransport,
			},
		},
	}
}

// TimedRoundTripper implements the RoundTripper interface.
type TimedRoundTripper struct {
	Transport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction and measures the duration.
func (t *TimedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	timer := prometheus.NewTimer(metrics.VaultRequestDuration)

	if t.Transport == nil {
		t.Transport = http.DefaultTransport
	}

	resp, err := t.Transport.RoundTrip(req)

	// handle response code and path
	metrics.VaultRequestErrorsTotal.With(prometheus.Labels{
		"response_code": strconv.Itoa(resp.StatusCode),
		"path":          req.URL.Path,
	}).Inc()

	timer.ObserveDuration()

	return resp, err
}
