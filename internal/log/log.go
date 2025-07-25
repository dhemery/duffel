// Package log configures logging for duffel.
package log

import (
	"fmt"
	"io"
	"log/slog"
)

var Level = &slog.LevelVar{}

var LevelNames = map[string]slog.Level{
	"none":  slog.LevelError + 4,
	"error": slog.LevelError,
	"warn":  slog.LevelWarn,
	"info":  slog.LevelInfo,
	"debug": slog.LevelDebug,
	"trace": slog.LevelDebug - 4,
}

func SetByName(level string, w io.Writer) error {
	l, ok := LevelNames[level]
	if !ok {
		return fmt.Errorf("invalid --log level: %s", level)
	}
	Set(l, w)
	return nil
}

func Set(l slog.Level, w io.Writer) {
	Level.Set(l)
	removeTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	ho := &slog.HandlerOptions{
		ReplaceAttr: removeTime,
		Level:       Level,
	}
	h := slog.NewJSONHandler(w, ho)
	slog.SetDefault(slog.New(h))
}
