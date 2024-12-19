package goweb

import (
	"context"
	"log/slog"
)

type discardLogHandler struct{}

func (h *discardLogHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h *discardLogHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *discardLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *discardLogHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return false
}
