package ilog

import (
	"app/conf"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLog() error {
	w := os.Stdout
	tFormat := "2006-01-02 15:04:05.000"

	consoleLevel := slog.LevelInfo
	if conf.C.Logging.LogDebug || conf.C.Logging.AlwaysShowDebugInConsole {
		consoleLevel = slog.LevelDebug
	}
	opts := &tint.Options{
		Level:      consoleLevel,
		TimeFormat: tFormat,
		AddSource:  true,
	}
	// Colorized console handler (with tint)
	consoleHandler := tint.NewHandler(colorable.NewColorable(w), opts)
	multiHandler := &MultiHandler{
		Handlers: []slog.Handler{consoleHandler},
	}

	ll := conf.C.Logging
	if !ll.Disabled {
		logRotator := &lumberjack.Logger{
			Filename:   path.Join(ll.LogDir, ll.FileName),
			MaxSize:    ll.FileSizeMB, // Max size in MB
			MaxBackups: ll.MaxBackups, // Number of backups
			Compress:   ll.Compress,   // Enable compression
		}

		lLevel := slog.LevelInfo
		if conf.C.Logging.LogDebug {
			lLevel = slog.LevelDebug
		}

		// Plain file handler (no color codes)
		fileHandler := tint.NewHandler(logRotator, &tint.Options{
			Level:      lLevel,
			TimeFormat: tFormat,
			AddSource:  true,
			NoColor:    ll.NoColor, // This disables color escape sequences
		})

		multiHandler.Handlers = append(multiHandler.Handlers, fileHandler)
	}

	if conf.C.Logging.ErrorLogging != nil && !conf.C.Logging.ErrorLogging.Disabled {
		logRotatorErr := &lumberjack.Logger{
			Filename:   path.Join(ll.ErrorLogging.LogDir, ll.ErrorLogging.FileName),
			MaxSize:    ll.ErrorLogging.FileSizeMB, // Max size in MB
			MaxBackups: ll.ErrorLogging.MaxBackups, // Number of backups
			Compress:   ll.ErrorLogging.Compress,   // Enable compression
		}

		fileHandlerErr := tint.NewHandler(logRotatorErr, &tint.Options{
			Level:      slog.LevelError,
			TimeFormat: tFormat,
			AddSource:  true,
			NoColor:    ll.NoColor, // This disables color escape sequences
		})
		multiHandler.Handlers = append(multiHandler.Handlers, fileHandlerErr)
	}

	var fileHandlerDbg slog.Handler
	if conf.C.Logging.DebugLogging != nil && !conf.C.Logging.DebugLogging.Disabled {
		logRotatorDbg := &lumberjack.Logger{
			Filename:   path.Join(ll.DebugLogging.LogDir, ll.DebugLogging.FileName),
			MaxSize:    ll.DebugLogging.FileSizeMB, // Max size in MB
			MaxBackups: ll.DebugLogging.MaxBackups, // Number of backups
			Compress:   ll.DebugLogging.Compress,   // Enable compression
		}

		fileHandlerDbg = tint.NewHandler(logRotatorDbg, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: tFormat,
			AddSource:  true,
			NoColor:    ll.NoColor, // This disables color escape sequences
		})
		multiHandler.Handlers = append(multiHandler.Handlers, fileHandlerDbg)
	}

	logger := slog.New(multiHandler)
	slog.SetDefault(logger)

	// Initialize web logger
	if err := InitWebLogger(fileHandlerDbg); err != nil {
		return fmt.Errorf("failed to initialize web logger: %w", err)
	}

	//do some test logging
	slog.Debug("Loggers initialized")
	slog.Info("Loggers initialized")
	slog.Warn("Loggers initialized")
	slog.Error("Loggers initialized")
	GetWebLogger().Info("Web logger initialized")
	return nil
}

func Err(err error) slog.Attr {
	if err != nil {
		return slog.Any("error", err)
	}
	return slog.String("error", "No-Error")
}
