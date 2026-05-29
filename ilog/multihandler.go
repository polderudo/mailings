package ilog

import (
	"context"
	"log/slog"
)

type MultiHandler struct {
	Handlers []slog.Handler
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.Handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range m.Handlers {
		// Check if this handler should handle this log level
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.Handlers))
	for i, h := range m.Handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{Handlers: handlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.Handlers))
	for i, h := range m.Handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{Handlers: handlers}
}
