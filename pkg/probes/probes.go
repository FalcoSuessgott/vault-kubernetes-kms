package probes

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Prober interface.
type Prober interface {
	Health() error
}

// HealthZ performs a health check for each prober and returns OK if all checks were successful.
func HealthZ(prober []Prober) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, p := range prober {
			if p == nil {
				return
			}

			if err := p.Health(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)

				zap.L().Error("health check failed", zap.Error(err))

				return
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, http.StatusText(http.StatusOK))

		zap.L().Debug("health checks succeeded")
	}
}
