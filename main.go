package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/exec"
	"github.com/dhemery/duffel/internal/file"
)

var (
	ErrLogLevel = errors.New("unknown log level")
	logLevel    = slog.LevelError
	logger      *slog.Logger
)

func main() {
	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	dryRunOpt := flags.Bool("n", false, "Print planned actions without executing them.")
	sourceOpt := flags.String("source", ".", "The source `dir`")
	targetOpt := flags.String("target", "..", "The target `dir`")
	flags.Func("log", "Set log level (default error)", setLogLevel)

	flags.Parse(os.Args[1:])

	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	root := "/"
	fsys := file.DirFS(root)

	absTarget, err := filepath.Abs(*targetOpt)
	if err != nil {
		fatal(fmt.Errorf("target: %w", err))
	}
	target, _ := filepath.Rel(root, absTarget)

	absSource, err := filepath.Abs(*sourceOpt)
	if err != nil {
		fatal(fmt.Errorf("source: %w", err))
	}
	source, _ := filepath.Rel(root, absSource)

	req := &exec.Request{
		FS:     fsys,
		Source: source,
		Target: target,
		Pkgs:   flags.Args(),
	}

	err = exec.Execute(req, *dryRunOpt, os.Stdout, logger)
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	logger.Error(err.Error())
	os.Exit(1)
}

func setLogLevel(name string) error {
	switch name {
	case "none":
		logLevel = slog.LevelError + 4
	case "error":
		logLevel = slog.LevelError
	case "warn":
		logLevel = slog.LevelWarn
	case "info":
		logLevel = slog.LevelInfo
	case "debug":
		logLevel = slog.LevelDebug
	default:
		return fmt.Errorf("%s: unknown log level; level must be one of: none, error, warn, info, debug", name)
	}
	return nil
}
