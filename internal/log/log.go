// Package log creates loggers.
package log

import (
	"io"
	"log/slog"
)

func NewJSONLogger(l slog.Level, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: l}))
}
