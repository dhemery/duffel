package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

type Options struct {
	source   string
	target   string
	dryRun   bool
	logLevel LogLevelOption
}

func ParseArgs(args []string) (Options, []string, error) {
	opts := Options{
		logLevel: LogLevelOption{slog.LevelError},
	}

	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	flags.BoolVar(&opts.dryRun, "n", false, "Print planned actions without executing them.")
	flags.Var(&opts.logLevel, "log", "Log `level`")
	flags.StringVar(&opts.source, "source", ".", "The source `dir`")
	flags.StringVar(&opts.target, "target", "..", "The target `dir`")
	err := flags.Parse(args)
	return opts, flags.Args(), err
}

type LogLevelOption struct {
	l slog.Level
}

func (o *LogLevelOption) String() string {
	return strings.ToLower(o.l.String())
}

func (o *LogLevelOption) Set(name string) error {
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

func (o *LogLevelOption) Level() slog.Level {
	return o.l
}
