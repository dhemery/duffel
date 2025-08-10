// Package cmd parses the command line and constructs a command
// to satisfy the user's request.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	"github.com/dhemery/duffel/internal/plan"
)

var (
	ErrLogLevel = errors.New("unknown log level")
)

type Planner interface {
	Plan([]*plan.PackageOp, *slog.Logger) (plan.Plan, error)
}

type Command struct {
	planner Planner
	FS      fs.FS
	DryRun  bool
}

func (c Command) Execute(ops []*plan.PackageOp, l *slog.Logger) error {
	plan, err := c.planner.Plan(ops, l)
	if err != nil {
		return err
	}

	if c.DryRun {
		return plan.Print(os.Stdout)
	}

	return plan.Execute(c.FS, l)
}

func Execute() error {
	var (
		dryRunOpt   bool
		logLevelOpt string
		sourceOpt   string
		targetOpt   string
	)
	flags := flag.NewFlagSet("duffel", flag.ExitOnError)
	flags.BoolVar(&dryRunOpt, "n", false, "Print planned actions without executing them.")
	flags.StringVar(&logLevelOpt, "log", "error", "Log level")
	flags.StringVar(&sourceOpt, "source", ".", "The source `dir`")
	flags.StringVar(&targetOpt, "target", "..", "The target `dir`")

	flags.Parse(os.Args[1:])

	root := "/"
	fsys := file.DirFS(root)

	target := mustRel("target", root, targetOpt)
	source := mustRel("source", root, sourceOpt)

	logger := log.Logger(os.Stderr, parseLogLevel(logLevelOpt))

	pkgOps := []*plan.PackageOp{}
	for _, pkg := range flags.Args() {
		pkgOp := plan.NewInstallOp(source, pkg)
		pkgOps = append(pkgOps, pkgOp)
	}

	c := Command{
		FS:      fsys,
		planner: plan.NewPlanner(fsys, target),
		DryRun:  dryRunOpt,
	}
	return c.Execute(pkgOps, logger)
}

func parseLogLevel(name string) slog.Level {
	switch name {
	case "none":
		return slog.LevelError + 4
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	default:
		fatal(fmt.Errorf("%s: unknown log level; level must be one of: none, error, warn, info, debug", name))
	}
	return 0
}

func fatal(e error) {
	fmt.Fprintln(os.Stderr, e.Error())
	os.Exit(1)
}

func mustRel(desc, root, name string) string {
	abs, err := filepath.Abs(name)
	if err != nil {
		fatal(fmt.Errorf("%q: %w", desc, err))
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		fatal(fmt.Errorf("%q: %w", desc, err))
	}
	return rel
}
