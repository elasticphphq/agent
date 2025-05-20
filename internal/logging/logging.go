package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/elasticphphq/agent/internal/config"
)

var logger *slog.Logger

func Init(cfg config.LoggingBlock) {
	var lvl slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: lvl,
			// TODO: Add color option once slog supports it natively
		})
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

func L() *slog.Logger {
	return logger
}
