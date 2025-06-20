package exec

import (
	"encoding/json"
	"io"
	"io/fs"
	"path/filepath"

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
	targetToSource, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return err
	}

	install := plan.Install{
		TargetToSource: targetToSource,
	}

	var pkgOps []plan.PkgOp
	for _, pkg := range r.Pkgs {
		pkgOps = append(pkgOps, plan.PkgOp{Pkg: pkg, Apply: install.Apply})
	}

	planner := plan.Planner{
		FS:     r.FS,
		Target: r.Target,
		Source: r.Source,
	}

	targetFS, err := fs.Sub(r.FS, r.Target)
	if err != nil {
		return err
	}
	stateLoader := file.StateLoader{FS: targetFS}

	plan, err := planner.Plan(pkgOps, stateLoader.Load)
	if err != nil {
		return err
	}

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}

	return plan.Execute(r.FS)
}
