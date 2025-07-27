package plan

import (
	"log/slog"
	"testing"
)

// SetTestLogger makes l the default [slog.Logger]
// and configures t to restore the prior default logger
// during cleanup.
func SetTestLogger(l *slog.Logger, t *testing.T) {
	priorLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(priorLogger) })
	slog.SetDefault(l)
}
