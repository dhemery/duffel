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
	stater := file.Stater{FS: r.FS}
	index := plan.NewIndex(stater)

	analyzer := plan.NewAnalyst(r.FS, index)
	pkgFinder := file.NewPkgFinder(r.FS)
	install := plan.NewInstallOp(r.Source, r.Target, pkgFinder, analyzer)

	var pkgOps []plan.PkgOp
	for _, pkg := range r.Pkgs {
		pkgOp := plan.NewPkgOp(r.Source, pkg, install)
		pkgOps = append(pkgOps, pkgOp)
	}

	planner := plan.NewPlanner(r.Target, analyzer)

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
