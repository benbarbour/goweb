package goweb

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type requestLogger struct {
	ctx     context.Context
	handler http.Handler
	logger  *slog.Logger
}

// ServeHTTP handles the request by passing it to the real
// handler and logging the request details
func (l *requestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w}
	l.handler.ServeHTTP(rec, r)
	l.logger.InfoContext(
		l.ctx,
		"request",
		"method", r.Method,
		"path", r.URL.Path,
		"rsp", rec.Status,
		"dur", time.Since(start),
	)
}

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

type discardHandler struct{}

func (h *discardHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h *discardHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *discardHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *discardHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return false
}
