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
	stater := file.DirStater{FS: r.FS, Dir: r.Target}
	index := plan.NewStateCache(stater)

	install := plan.Install{Source: r.Source, Target: r.Target}

	var pkgOps []plan.PkgOp
	for _, pkg := range r.Pkgs {
		pkgOp := plan.PkgOp{Source: r.Source, Pkg: pkg, ItemOp: install, Index: index}
		pkgOps = append(pkgOps, pkgOp)
	}

	planner := plan.Planner{
		Target:   r.Target,
		Analyzer: plan.Analyst{FS: r.FS},
		States:   index,
	}

	plan, err := planner.Plan(pkgOps)
	if err != nil {
		return err
	}

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}

	return plan.Execute(r.FS)
}
