package ilog

import (
	"log/slog"

	"github.com/labstack/echo/v4"
)

// NewEchoContextLogger creates a new slog logger and adds the Request ID if it exists in the Echo context
// Usage: logger := ilog.NewContextLogger(c)
func NewEchoContextLogger(c echo.Context) *slog.Logger {
	logger := slog.Default()

	if c != nil {
		// Check if Request ID middleware is being used and get the request ID
		if requestID := GetRequestID(c); requestID != "" {
			logger = logger.With("request_id", requestID)
		}
	}
	return logger
}

// NewEchoContextLoggerWithAttrs creates a new slog logger with Request ID and additional attributes
// Usage: logger := ilog.NewContextLoggerWithAttrs(c, "user_id", userID, "action", "create")
func NewEchoContextLoggerWithAttrs(c echo.Context, attrs ...any) *slog.Logger {
	logger := slog.Default()

	// Start with request ID if available
	if requestID := GetRequestID(c); requestID != "" {
		attrs = append([]any{"request_id", requestID}, attrs...)
	}

	if len(attrs) > 0 {
		logger = logger.With(attrs...)
	}

	return logger
}

// GetRequestID attempts to get the request ID from various common middleware setups
func GetRequestID(c echo.Context) string {
	// Try Echo's built-in request ID middleware header key
	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		return requestID
	}

	// Try getting from request header
	if requestID := c.Request().Header.Get(echo.HeaderXRequestID); requestID != "" {
		return requestID
	}

	// Try getting from context values using common keys
	// Echo's RequestID middleware typically stores it with this key
	if requestID := c.Get("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	// Try alternative context key names
	if requestID := c.Get("requestId"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	if requestID := c.Get("x-request-id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	// Try common alternative header names
	if requestID := c.Request().Header.Get("X-Trace-Id"); requestID != "" {
		return requestID
	}

	if requestID := c.Request().Header.Get("X-Correlation-ID"); requestID != "" {
		return requestID
	}

	return ""
}

// WithEchoRequestID is a convenience function to add request ID to an existing logger
func WithEchoRequestID(logger *slog.Logger, c echo.Context) *slog.Logger {
	if requestID := GetRequestID(c); requestID != "" {
		return logger.With("request_id", requestID)
	}
	return logger
}
