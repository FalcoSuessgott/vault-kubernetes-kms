package probes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type ErrorProber struct{}

func (p *ErrorProber) Health() error { return errors.New("probe failed") }

type SuccessProber struct{}

func (p *SuccessProber) Health() error { return nil }

func TestHealthZ(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		prober := []Prober{&ErrorProber{}}

		hf := HealthZ(prober)

		req := httptest.NewRequest(http.MethodGet, "https://google.de", nil)
		w := httptest.NewRecorder()
		hf(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(t, "probe failed", w.Body.String())
	})

	t.Run("succcess", func(t *testing.T) {
		prober := []Prober{&SuccessProber{}}

		hf := HealthZ(prober)

		req := httptest.NewRequest(http.MethodGet, "https://google.de", nil)
		w := httptest.NewRecorder()
		hf(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, http.StatusText(http.StatusOK), w.Body.String())
	})
}
