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
	index := plan.NewStateCache(stater)

	install := plan.NewInstallOp(r.Source, r.Target, nil)

	var pkgOps []plan.PkgOp
	for _, pkg := range r.Pkgs {
		pkgOp := plan.NewPkgOp(r.Source, pkg, install, index)
		pkgOps = append(pkgOps, pkgOp)
	}

	analyzer := plan.NewAnalyzer(r.FS)

	planner := plan.NewPlanner(r.Target, analyzer, index)

	plan, err := planner.Plan(pkgOps...)
	if err != nil {
		return err
	}

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}

	return plan.Execute(r.FS)
}
