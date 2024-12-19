package goweb

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func MW_RequestLogger(ctx context.Context, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		logger.InfoContext(
			ctx,
			"request",
			"method", r.Method,
			"path", r.URL.Path,
			"rsp", rec.Status,
			"dur", time.Since(start),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}
