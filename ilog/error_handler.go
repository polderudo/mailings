package ilog

import (
	"context"
	"log/slog"
)

// ErrorOnlyHandler wraps a slog.Handler to only process ERROR level and above
type ErrorOnlyHandler struct {
	handler slog.Handler
}

func (h *ErrorOnlyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelError && h.handler.Enabled(ctx, level)
}

func (h *ErrorOnlyHandler) Handle(ctx context.Context, record slog.Record) error {
	// Only handle if it's an error or above
	if record.Level >= slog.LevelError {
		return h.handler.Handle(ctx, record)
	}
	return nil
}

func (h *ErrorOnlyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ErrorOnlyHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

func (h *ErrorOnlyHandler) WithGroup(name string) slog.Handler {
	return &ErrorOnlyHandler{
		handler: h.handler.WithGroup(name),
	}
}
