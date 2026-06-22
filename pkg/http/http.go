package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
)

const requestTimeout = 10 * time.Second

// RoundTripper is a custom HTTP RoundTripper.
type RoundTripper struct {
	Transport http.RoundTripper
}

func (rd *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	resp, err := rd.Transport.RoundTrip(req)

	status := "error"
	if resp != nil {
		status = strconv.Itoa(resp.StatusCode)
	}

	metrics.VaultRequestsDurationSeconds.WithLabelValues(req.Method, req.URL.Path, status).Observe(time.Since(startTime).Seconds())

	return resp, err
}

// New returns a custom http client backed by a clone of the default transport.
func New() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	return &http.Client{
		Timeout: requestTimeout,
		Transport: &RoundTripper{
			Transport: transport.Clone(),
		},
	}
}

// NewWithTransport wraps an existing transport with the metrics round tripper,
// preserving any TLS configuration already present on the transport.
func NewWithTransport(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Timeout:   requestTimeout,
		Transport: &RoundTripper{Transport: transport},
	}
}
