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

const (
	executionError = iota + 1
	badUsage
)

var (
	ErrLogLevel = errors.New("unknown log level")
)

// FS is a [fs.FS] that implements all of the methods used by duffel.
type FS interface {
	fs.ReadLinkFS
	plan.ActionFS
	Name() string
}

// FSFunc is a function that returns a [FS].
type FSFunc func() (FS, error)

// Execute parses and validates args, then executes the user's request
// against the [FS] returned by fsFunc.
func Execute(args []string, fsFunc FSFunc, wout, werr io.Writer) {
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

	if err := flags.Parse(args); err != nil {
		fatalUsage(werr, err)
	}

	fsys, err := fsFunc()
	if err != nil {
		fatal(werr, err)
	}

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
	if err = c.Execute(pkgOps, logger); err != nil {
		fatal(werr, err)
	}
}

// A Planner creates a [plan.Plan] to implement a sequence of [*plan.PackageOp].
type Planner interface {
	// Plan creates a [plan.Plan] to implement the ops.
	Plan(ops []*plan.PackageOp, l *slog.Logger) (plan.Plan, error)
}

// A Command creates and executes a plan to implement a sequence of [*plan.PackageOp].
type Command struct {
	planner Planner
	FS      FS
	Out     io.Writer
	DryRun  bool
}

// Execute creates and executes a plan to implement ops.
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
		err := fmt.Errorf("%s: unknown log level; level must be one of: none, error, warn, info, debug", name)
		fatalUsage(werr, err)
	}
	return 0
}

func fatal(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(1)
}

func fatalUsage(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(2)
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
