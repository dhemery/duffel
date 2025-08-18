// Package cmd constructs and executes a plan
// to satisfy the user's request.
package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/dhemery/duffel/internal/plan"
)

const Root = "/"

// FS is an [fs.FS] that implements all of the methods used by duffel.
type FS interface {
	fs.ReadLinkFS
	plan.ActionFS
}

// FSFunc creates an [FS] rooted at root.
type FSFunc func(root string) (FS, error)

// Execute performs the duffel operations requested by args.
func Execute(args []string, fsFunc FSFunc, wout, werr io.Writer) {
	fsys, err := fsFunc(Root)
	if err != nil {
		fatal(werr, err)
	}

	cmd, err := Compile(args, fsys, wout, werr)
	if err != nil {
		fatalUsage(werr, err)
	}

	if err := cmd.Execute(); err != nil {
		fatal(werr, err)
	}
}

// Command creates a [plan.Plan] and acts on it.
type Command struct {
	Planner  Planner  // Creates the plan.
	PlanFunc PlanFunc // Acts on the plan.
}

// Execute creates a plan and acts on it.
func (c Command) Execute() error {
	plan, err := c.Planner.Plan()
	if err != nil {
		return err
	}

	return c.PlanFunc(plan)
}

// A Planner creates a [plan.Plan].
type Planner interface {
	// Plan creates a [plan.Plan].
	Plan() (plan.Plan, error)
}

// PlanFunc acts on a [plan.Plan].
type PlanFunc func(p plan.Plan) error

func fatal(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(1)
}

func fatalUsage(w io.Writer, e error) {
	fmt.Fprintln(w, e.Error())
	os.Exit(2)
}
