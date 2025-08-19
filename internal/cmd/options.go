package cmd

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"strings"
)

// Options provides the set of options parsed from the command arguments.
type Options struct {
	Source   string
	Target   string
	DryRun   bool
	LogLevel slog.Level
}

var (
	OptDefaultSource   = "."
	OptDefaultTarget   = ".."
	OptDefaultDryRun   = false
	OptDefaultLogLevel = slog.LevelError
	ErrLogLevel        = errors.New("must be one of none, error, warn, info, debug")
)

// ParseArgs returns the [Options] parsed from args.
// The []string result holds the non-flag args.
func ParseArgs(args []string, werr io.Writer) (Options, []string, error) {
	opts := Options{LogLevel: OptDefaultLogLevel}
	logLevelOpt := &LogLevelValue{&opts.LogLevel}

	flags := flag.NewFlagSet("duffel", flag.ContinueOnError)
	flags.SetOutput(werr)

	flags.BoolVar(&opts.DryRun, "n", OptDefaultDryRun, "Print planned actions without executing them")
	flags.Var(logLevelOpt, "log", "Log `level`")
	flags.StringVar(&opts.Source, "source", OptDefaultSource, "The source `dir`")
	flags.StringVar(&opts.Target, "target", OptDefaultTarget, "The target `dir`")

	err := flags.Parse(args)

	return opts, flags.Args(), err
}

// LogLevelValue is the minimum severity level for duffel to log.
type LogLevelValue struct {
	Level *slog.Level
}

// String implements [flag.Value].
func (v *LogLevelValue) String() string {
	if v.Level == nil {
		return "<nil>"
	}
	return strings.ToLower(v.Level.String())
}

// Set implements [flag.Value].
func (v *LogLevelValue) Set(name string) error {
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
		return ErrLogLevel
	}
	return nil
}
