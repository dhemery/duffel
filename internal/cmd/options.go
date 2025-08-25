package cmd

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"strings"
)

// options provides the set of options parsed from the command arguments.
type options struct {
	source   string
	target   string
	dryRun   bool
	logLevel slog.Level
}

var (
	optDefaultSource   = "."
	optDefaultTarget   = ".."
	optDefaultDryRu    = false
	optDefaultLogLevel = slog.LevelError
	errLogLevel        = errors.New("must be one of none, error, warn, info, debug")
)

// parseArgs returns the [options] parsed from args.
// The []string result holds the non-flag args.
func parseArgs(args []string, werr io.Writer) (options, []string, error) {
	opts := options{logLevel: optDefaultLogLevel}
	logLevelOpt := &logLevelValue{&opts.logLevel}

	flags := flag.NewFlagSet("duffel", flag.ContinueOnError)
	flags.SetOutput(werr)

	flags.BoolVar(&opts.dryRun, "n", optDefaultDryRu, "Print planned actions without executing them")
	flags.Var(logLevelOpt, "log", "Log `level`")
	flags.StringVar(&opts.source, "source", optDefaultSource, "The source `dir`")
	flags.StringVar(&opts.target, "target", optDefaultTarget, "The target `dir`")

	err := flags.Parse(args)

	return opts, flags.Args(), err
}

// logLevelValue is the minimum severity level for duffel to log.
type logLevelValue struct {
	Level *slog.Level
}

// String implements [flag.Value].
func (v *logLevelValue) String() string {
	if v.Level == nil {
		return "<nil>"
	}
	return strings.ToLower(v.Level.String())
}

// Set implements [flag.Value].
func (v *logLevelValue) Set(name string) error {
	switch name {
	case "none":
		*v.Level = slog.LevelError + 4
	case "error":
		*v.Level = slog.LevelError
	case "warn":
		*v.Level = slog.LevelWarn
	case "info":
		*v.Level = slog.LevelInfo
	case "debug":
		*v.Level = slog.LevelDebug
	default:
		return errLogLevel
	}
	return nil
}
