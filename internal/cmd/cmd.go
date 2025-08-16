// Package cmd parses the command line and constructs a command
// to satisfy the user's request.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dhemery/duffel/internal/log"
	"github.com/dhemery/duffel/internal/plan"
)

var (
	ErrLogLevel = errors.New("unknown log level")
)

type FS interface {
	fs.ReadLinkFS
	plan.ActionFS
	Name() string
}

type Planner interface {
	Plan([]*plan.PackageOp, *slog.Logger) (plan.Plan, error)
}

type Command struct {
	planner Planner
	FS      plan.ActionFS
	Out     io.Writer
	DryRun  bool
}

func (c Command) Execute(ops []*plan.PackageOp, l *slog.Logger) error {
	plan, err := c.planner.Plan(ops, l)
	if err != nil {
		return err
	}

	if c.DryRun {
		return plan.Print(c.Out)
	}

	return plan.Execute(c.FS, l)
}

func Execute(args []string, fsys FS, wout, werr io.Writer) error {
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

	flags.Parse(args)

	target := mustRel("target", fsys.Name(), targetOpt, werr)
	source := mustRel("source", fsys.Name(), sourceOpt, werr)

	logger := log.Logger(werr, parseLogLevel(logLevelOpt, werr))

	pkgOps := []*plan.PackageOp{}
	for _, pkg := range flags.Args() {
		pkgOp := plan.NewInstallOp(source, pkg)
		pkgOps = append(pkgOps, pkgOp)
	}

	c := Command{
		FS:      fsys,
		Out:     wout,
		planner: plan.NewPlanner(fsys, target),
		DryRun:  dryRunOpt,
	}
	return c.Execute(pkgOps, logger)
}

func parseLogLevel(name string, werr io.Writer) slog.Level {
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
		fatal(werr, fmt.Errorf("%s: unknown log level; level must be one of: none, error, warn, info, debug", name))
	}
	return 0
}

func fatal(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(1)
}

func mustRel(desc, root, name string, w io.Writer) string {
	abs, err := filepath.Abs(name)
	if err != nil {
		fatal(w, fmt.Errorf("%q: %w", desc, err))
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		fatal(w, fmt.Errorf("%q: %w", desc, err))
	}
	return rel
}
