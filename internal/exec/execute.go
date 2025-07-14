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
	stater := file.NewStater(r.FS, r.Target)
	index := plan.NewIndex(stater)

	analyzer := plan.NewAnalyst(r.FS, index)
	merger := plan.NewMerger(nil, analyzer)
	install := plan.NewInstallOp(r.Source, r.Target, merger)

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
