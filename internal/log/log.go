// Package log configures logging for duffel.
package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

const (
	LevelNone  = slog.LevelError + 4
	LevelError = slog.LevelError
	LevelWarn  = slog.LevelWarn
	LevelInfo  = slog.LevelInfo
	LevelDebug = slog.LevelDebug
	LevelTrace = slog.LevelDebug - 4
)

var Level = &slog.LevelVar{}

var LevelNames = map[string]slog.Level{
	"none":  LevelNone,
	"error": LevelError,
	"warn":  LevelWarn,
	"info":  LevelInfo,
	"debug": LevelDebug,
	"trace": LevelTrace,
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

var ctx = context.TODO()

func Error(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, LevelError, msg, attrs...)
}

func Warn(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, LevelWarn, msg, attrs...)
}

func Info(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, LevelInfo, msg, attrs...)
}

func Debug(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, LevelDebug, msg, attrs...)
}

func Trace(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, LevelTrace, msg, attrs...)
}
