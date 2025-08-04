package duftest

import (
	"flag"
	"fmt"
	"log/slog"
)

var LogLevel slog.Level

func init() {
	flag.Func("log", "log level (default info)", setLogLevel)
}

func setLogLevel(name string) error {
	switch name {
	case "none":
		LogLevel = slog.LevelError + 4
	case "error":
		LogLevel = slog.LevelError
	case "warn":
		LogLevel = slog.LevelWarn
	case "info":
		LogLevel = slog.LevelInfo
	case "debug":
		LogLevel = slog.LevelDebug
	default:
		return fmt.Errorf("%s: unknown log level; level must be one of: none, error, warn, info, debug", name)
	}
	return nil
}
