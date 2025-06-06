package duftest

import (
	"io"
	"log/slog"
)

func omitTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}

func Logger(name string, level slog.Level, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: omitTime,
	}
	h := slog.NewJSONHandler(w, opts)

	return slog.New(h).With("test", name)
}
