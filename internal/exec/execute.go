package exec

import (
	"encoding/json"
	"io"
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/plan"
)

type Request struct {
	FS     fs.FS
	Source string
	Target string
	Pkgs   []string
}

func Execute(r *Request, dryRun bool, w io.Writer) error {
	install := plan.Install{Source: r.Source, Target: r.Target}

	var pkgOps []plan.PkgOp
	for _, pkg := range r.Pkgs {
		pkgOps = append(pkgOps, plan.PkgOp{Pkg: pkg, ItemOp: install})
	}

	planner := plan.Planner{
		FS:     r.FS,
		Target: r.Target,
		Source: r.Source,
	}

	stater := file.DirStater{FS: r.FS, Dir: r.Target}

	plan, err := planner.Plan(pkgOps, stater)
	if err != nil {
		return err
	}

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}

	return plan.Execute(r.FS)
}
