package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
)

// RoundTripper is a custom http rountripper.
type RoundTripper struct {
	Transport http.RoundTripper
}

func (rd *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	resp, err := rd.Transport.RoundTrip(req)

	endTime := time.Now()

	// nolnt: perfsprint
	metrics.VaultRequestsDurationSeconds.WithLabelValues(req.Method, req.URL.Path, fmt.Sprint(resp.StatusCode)).Observe(float64(endTime.Sub(startTime).Seconds()))

	return resp, err
}

// New returns a custom http client.
func New() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &RoundTripper{
			Transport: &http.Transport{},
		},
	}
}
