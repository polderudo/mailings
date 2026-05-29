package ilog

import (
	"app/conf"
	"context"
	"io"
	"log/slog"
	"os"
	"path"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"gopkg.in/natefinch/lumberjack.v2"
)

var webLogger *slog.Logger

// InitWebLogger initializes the web logger that logs to web.log
func InitWebLogger(dbgHandler slog.Handler) error {
	tFormat := "2006-01-02 15:04:05.000"

	ll := conf.C.Logging

	// Create file handler for web.log
	webLogRotator := &lumberjack.Logger{
		Filename:   path.Join(ll.WebLogging.LogDir, ll.WebLogging.FileName),
		MaxSize:    ll.WebLogging.FileSizeMB, // Max size in MB
		MaxBackups: ll.WebLogging.MaxBackups, // Number of backups
		Compress:   ll.WebLogging.Compress,   // Enable compression
	}

	// Create file handler
	fileHandler := tint.NewHandler(webLogRotator, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: tFormat,
		AddSource:  false, // Usually don't need source for web requests
		NoColor:    true,  // No colors in file logs
	})

	var handlers []slog.Handler
	handlers = append(handlers, fileHandler)

	// If LogWebToConsole is true, also log to the default slog logger
	if conf.C.LogWebToConsole {
		// Create a console handler specifically without source info
		consoleHandler := &NoSourceConsoleHandler{
			handler: tint.NewHandler(colorable.NewColorable(os.Stdout), &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: tFormat,
				AddSource:  false, // Explicitly disable source for console forwarding
			}),
		}
		handlers = append(handlers, consoleHandler)
	}

	// Create multi-handler
	multiHandler := &MultiHandler{
		Handlers: handlers,
	}

	if dbgHandler != nil && conf.C.LogWebToDebug {
		multiHandler.Handlers = append(multiHandler.Handlers, dbgHandler)
	}

	// Create the web logger with web-specific attributes
	webLogger = slog.New(multiHandler).With(
		"type", "web",
		"component", "echo",
	)

	return nil
}

// GetWebLogger returns the web logger instance
func GetWebLogger() *slog.Logger {
	if webLogger == nil {
		// Fallback to default logger if not initialized
		return slog.Default().With("type", "web", "component", "echo")
	}
	return webLogger
}

// NoSourceConsoleHandler wraps a handler and strips source information
type NoSourceConsoleHandler struct {
	handler slog.Handler
}

func (h *NoSourceConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *NoSourceConsoleHandler) Handle(ctx context.Context, record slog.Record) error {
	// Create a new record without source information
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)

	// Copy all attributes except source-related ones
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key != slog.SourceKey {
			newRecord.AddAttrs(attr)
		}
		return true
	})

	// Set PC to 0 to prevent source information from being added
	newRecord.PC = 0

	return h.handler.Handle(ctx, newRecord)
}

func (h *NoSourceConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Filter out source attributes
	filteredAttrs := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Key != slog.SourceKey {
			filteredAttrs = append(filteredAttrs, attr)
		}
	}

	return &NoSourceConsoleHandler{
		handler: h.handler.WithAttrs(filteredAttrs),
	}
}

func (h *NoSourceConsoleHandler) WithGroup(name string) slog.Handler {
	return &NoSourceConsoleHandler{
		handler: h.handler.WithGroup(name),
	}
}

// NewWebRequestLogger creates a logger for a specific web request with request-specific attributes
func NewWebRequestLogger(method, path, requestID string) *slog.Logger {
	logger := GetWebLogger()
	attrs := []any{
		"method", method,
		"path", path,
	}

	if requestID != "" {
		attrs = append(attrs, "request_id", requestID)
	}

	return logger.With(attrs...)
}

// WebLogWriter creates an io.Writer that writes to the web logger
// This can be used directly with Echo's logger
func WebLogWriter() io.Writer {
	return &WebLogWriterAdapter{
		logger: GetWebLogger(),
	}
}

// WebLogWriterAdapter adapts the web logger to implement io.Writer
type WebLogWriterAdapter struct {
	logger *slog.Logger
}

func (w *WebLogWriterAdapter) Write(p []byte) (n int, err error) {
	// Remove trailing newline if present
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	// Log the message at Info level
	w.logger.Info(msg)

	return len(p), nil
}
