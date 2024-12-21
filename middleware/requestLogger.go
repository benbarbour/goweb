package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func RequestLogger(logger *slog.Logger, lvl slog.Level, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var reqLogger *slog.Logger
		srvID := uuid.NewString()
		cliID := r.Header.Get("X-Request-ID")
		if cliID != "" {
			reqLogger = logger.With(slog.Group("req",
				slog.String("srvID", srvID),
				slog.String("cliID", cliID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			))
		} else {
			reqLogger = logger.With(slog.Group("req",
				slog.String("srvID", srvID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			))
		}
		ctx := context.WithValue(r.Context(), ctxKeyLogger, reqLogger)

		reqLogger.Log(ctx, lvl, "HTTP req")
		r = r.WithContext(ctx)
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		reqLogger.Log(
			ctx, lvl,
			"HTTP rsp",
			"status", rec.Status,
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

func GetLoggerFromCtx(ctx context.Context) *slog.Logger {
	v := ctx.Value(ctxKeyLogger)
	if l, ok := v.(*slog.Logger); ok {
		return l
	}
	return nil
}
