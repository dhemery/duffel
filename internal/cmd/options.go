package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

// Options provides the set of options parsed from the command arguments.
type Options struct {
	source   string
	target   string
	dryRun   bool
	logLevel LogLevelOpt
}

// ParseArgs returns the [Options] parsed from args.
// The []string result holds the non-flag args.
func ParseArgs(args []string) (Options, []string, error) {
	opts := Options{
		logLevel: LogLevelOpt{slog.LevelError},
	}

	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	flags.BoolVar(&opts.dryRun, "n", false, "Print planned actions without executing them.")
	flags.Var(&opts.logLevel, "log", "Log `level`")
	flags.StringVar(&opts.source, "source", ".", "The source `dir`")
	flags.StringVar(&opts.target, "target", "..", "The target `dir`")
	err := flags.Parse(args)
	return opts, flags.Args(), err
}

// LogLevelOpt is the minimum severity level for duffel to log.
type LogLevelOpt struct {
	l slog.Level
}

// String implements [flag.Value].
func (o *LogLevelOpt) String() string {
	return strings.ToLower(o.l.String())
}

// Set implements [flag.Value].
func (o *LogLevelOpt) Set(name string) error {
	switch name {
	case "none":
		o.l = slog.LevelError + 4
	case "error":
		o.l = slog.LevelError
	case "warn":
		o.l = slog.LevelWarn
	case "info":
		o.l = slog.LevelInfo
	case "debug":
		o.l = slog.LevelDebug
	default:
		return fmt.Errorf("%s: unknown log level; level must be one of: none, error, warn, info, debug", name)
	}
	return nil
}

// Level implements [slog.Leveler].
func (o *LogLevelOpt) Level() slog.Level {
	return o.l
}
