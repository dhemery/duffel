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
	"path"
	"path/filepath"
	"strings"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	"github.com/dhemery/duffel/internal/plan"
)

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

type options struct {
	source   string
	target   string
	dryRun   bool
	logLevel LogLevelOption
}

func Parse(args []string) (options, []string, error) {
	opts := options{
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

// FS is a [fs.FS] that implements all of the methods used by duffel.
type FS interface {
	fs.ReadLinkFS
	plan.ActionFS
}

// FSFunc is a function that returns a [FS] rooted at root.
type FSFunc func(root string) (FS, error)

// Execute parses and validates args, then executes the user's request
// against the [FS] returned by fsFunc.
func Execute(args []string, fsFunc FSFunc, wout, werr io.Writer) {
	opts, args, err := Parse(args)
	if err != nil {
		fatalUsage(werr, err)
	}

	root := "/"
	target := mustRel("target", root, opts.target, werr)
	source := mustRel("source", root, opts.source, werr)

	pkgOps := []*plan.PackageOp{}
	for _, pkg := range args {
		pkgOp := plan.NewInstallOp(source, pkg)
		pkgOps = append(pkgOps, pkgOp)
	}

	fsys, err := fsFunc(root)
	if err != nil {
		fatal(werr, err)
	}

	if err = validate(fsys, target, source, pkgOps); err != nil {
		fatal(werr, err)
	}

	c := Command{
		FS:      fsys,
		Out:     wout,
		planner: plan.NewPlanner(fsys, target),
		DryRun:  opts.dryRun,
	}

	logger := log.Logger(werr, &opts.logLevel)
	if err = c.Execute(pkgOps, logger); err != nil {
		fatal(werr, err)
	}
}

func validate(fsys FS, target, source string, pkgOps []*plan.PackageOp) error {
	terr := validateDir(fsys, "target", target)
	serr := validateSource(fsys, source)
	errs := []error{terr, serr}

	for _, op := range pkgOps {
		operr := validatePackage(op)
		if operr == nil && serr == nil {
			operr = validateDir(fsys, "package", op.Path())
		}
		errs = append(errs, operr)
	}

	return errors.Join(errs...)
}

func validateDir(fsys FS, ctx, name string) error {
	info, err := fsys.Lstat(name)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx, err)
	}

	if !info.IsDir() {
		typ, _ := file.TypeOf(info.Mode().Type())
		return fmt.Errorf("%s %s (type %s): not a directory",
			ctx, name, typ)
	}

	return nil
}

func validateSource(fsys FS, source string) error {
	if err := validateDir(fsys, "source", source); err != nil {
		return err
	}
	return nil
}

func validatePackage(op *plan.PackageOp) error {
	packageDir := path.Dir(op.Path())
	if packageDir != op.Source() {
		return fmt.Errorf("package %s: not a child of source %s", op.Package(), op.Source())
	}

	return nil
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
