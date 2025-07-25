package exec

import (
	"encoding/json"
	"io"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/plan"
)

type Request struct {
	FS     fs.FS
	Source string
	Target string
	Pkgs   []string
}

func Execute(r *Request, dryRun bool, w io.Writer) error {
	stater := plan.NewStater(r.FS)
	index := plan.NewIndex(stater)

	pkgFinder := plan.NewPkgFinder(r.FS)
	analyzer := plan.NewAnalyst(r.FS, index)
	merger := plan.NewMerger(pkgFinder, analyzer)
	install := plan.NewInstallOp(r.Source, r.Target, merger)

	var pkgOps []plan.PkgOp
	for _, pkg := range r.Pkgs {
		sourcePkg := path.Join(r.Source, pkg)
		pkgOp := plan.NewPkgOp(sourcePkg, install)
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
