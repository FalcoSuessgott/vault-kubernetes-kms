package probes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type probeContextKey struct{}

type ErrorProber struct{}

func (p *ErrorProber) Health(ctx context.Context) error { return errors.New("probe failed") }

type SuccessProber struct{}

func (p *SuccessProber) Health(ctx context.Context) error { return nil }

type ContextProber struct {
	value any
}

func (p *ContextProber) Health(ctx context.Context) error {
	p.value = ctx.Value(probeContextKey{})

	return nil
}

type CancelledContextProber struct{}

func (p *CancelledContextProber) Health(ctx context.Context) error {
	<-ctx.Done()

	return ctx.Err()
}

func TestHealthZ(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		prober := []Prober{&ErrorProber{}}

		hf := HealthZ(prober)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://google.de", nil)
		w := httptest.NewRecorder()
		hf(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(t, "probe failed", w.Body.String())
	})

	t.Run("succcess", func(t *testing.T) {
		prober := []Prober{&SuccessProber{}}

		hf := HealthZ(prober)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://google.de", nil)
		w := httptest.NewRecorder()
		hf(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, http.StatusText(http.StatusOK), w.Body.String())
	})

	t.Run("passes request context", func(t *testing.T) {
		prober := &ContextProber{}
		hf := HealthZ([]Prober{prober})

		ctx := context.WithValue(context.Background(), probeContextKey{}, "value")
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "https://google.de", nil)
		w := httptest.NewRecorder()
		hf(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "value", prober.value)
	})

	t.Run("cancelled request context is propagated", func(t *testing.T) {
		hf := HealthZ([]Prober{&CancelledContextProber{}})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "https://google.de", nil)
		w := httptest.NewRecorder()
		hf(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(t, context.Canceled.Error(), w.Body.String())
	})
}
