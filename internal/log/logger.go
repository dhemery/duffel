// Package log provides logging features for duffel.
package log

import (
	"io"
	"log/slog"
)

func Logger(w io.Writer, l slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: l, ReplaceAttr: DiscardTimeAttr}
	handler := slog.NewTextHandler(w, opts)
	return slog.New(handler)
}

func DiscardTimeAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}
