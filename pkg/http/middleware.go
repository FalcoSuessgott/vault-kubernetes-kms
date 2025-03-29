package http

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)

		zap.L().Debug("received request",
			zap.String("method", r.Method),
			zap.String("path", r.RequestURI),
			zap.String("duration", time.Since(start).String()),
			zap.String("client", r.RemoteAddr),
		)
	})
}
