package mw

import (
	"app/ilog"
	"time"

	"github.com/labstack/echo/v4"
)

func WebLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		// Get request ID if available
		requestID := ilog.GetRequestID(c) // Using the function from context_logger.go

		// Create request-specific logger
		reqLogger := ilog.NewWebRequestLogger(
			c.Request().Method,
			c.Request().URL.Path,
			requestID,
		)

		// Store logger in context for use in handlers
		c.Set("logger", reqLogger)

		err := next(c)

		// Log request completion
		duration := time.Since(start)
		status := c.Response().Status

		logAttrs := []any{
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"bytes_out", c.Response().Size,
		}

		if err != nil {
			logAttrs = append(logAttrs, "error", err.Error())
			reqLogger.Error("Request completed with error", logAttrs...)
		} else if c.Request().Method != "OPTIONS" {
			reqLogger.Info("Request completed", logAttrs...)
		}

		return err
	}
}
