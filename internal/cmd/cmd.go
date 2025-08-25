// Package cmd constructs and executes a plan
// to satisfy the user's request.
package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/plan"
)

// FS is an [fs.FS] that implements all of the methods used by duffel.
type FS interface {
	fs.ReadLinkFS
	file.ActionFS
}

// Execute performs the duffel operations requested by args.
func Execute(args []string, fsys FS, cwd string, wout, werr io.Writer) {
	opts, args, err := ParseArgs(args, werr)
	if err != nil {
		fatalUsage(werr, err)
	}

	cmd, err := Compile(opts, args, fsys, cwd[1:], wout, werr)
	if err != nil {
		fatalUsage(werr, err)
	}

	if err := cmd.execute(); err != nil {
		fatal(werr, err)
	}
}

// A command creates a [plan.Plan] and acts on it.
type command struct {
	Planner  planner  // Creates the plan.
	PlanFunc planFunc // Acts on the plan.
}

// execute creates a plan and acts on it.
func (c command) execute() error {
	plan, err := c.Planner.Plan()
	if err != nil {
		return err
	}

	return c.PlanFunc(plan)
}

// A planner creates a [plan.Plan].
type planner interface {
	// Plan creates a [plan.Plan].
	Plan() (plan.Plan, error)
}

// planFunc acts on a [plan.Plan].
type planFunc func(p plan.Plan) error

func fatal(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(1)
}

func fatalUsage(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(2)
}
