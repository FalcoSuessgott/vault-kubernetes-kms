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

// New returns a custom http client.
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
