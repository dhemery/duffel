// Package cmd parses the command line and constructs a command
// to satisfy the user's request.
package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/dhemery/duffel/internal/log"
	"github.com/dhemery/duffel/internal/plan"
)

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
	root := "/"
	fsys, err := fsFunc(root)
	if err != nil {
		fatal(werr, err)
	}

	opts, args, err := ParseArgs(args)
	if err != nil {
		fatalUsage(werr, err)
	}

	req, err := CompileRequest(root, fsys, opts, args)
	if err != nil {
		fatal(werr, err)
	}

	c := Command{
		FS:      fsys,
		Out:     wout,
		planner: plan.NewPlanner(fsys, req.target),
		DryRun:  opts.dryRun,
	}

	logger := log.Logger(werr, &opts.logLevel)
	if err := c.Execute(req.ops, logger); err != nil {
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

func fatal(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(1)
}

func fatalUsage(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(2)
}
