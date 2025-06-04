package duffel

import (
	"encoding/json"
	"io"
	"path/filepath"
)

func Execute(r *Request, dryRun bool, w io.Writer) error {
	targetToSource, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return err
	}

	image := Image{}
	install := Install{
		source:         r.Source,
		target:         r.Target,
		targetToSource: targetToSource,
		image:          image,
	}

	var pkgAnalysts []PkgAnalyst
	for _, pkg := range r.Pkgs {
		pa := NewPkgAnalyst(r.FS, r.Source, pkg, install)
		pkgAnalysts = append(pkgAnalysts, pa)
	}

	for _, pa := range pkgAnalysts {
		err = pa.Analyze()
		if err != nil {
			break
		}
	}

	tasks := image.Tasks()

	plan := Plan{Target: r.Target, Tasks: tasks}

	if dryRun {
		enc := json.NewEncoder(w)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}

	return plan.Execute(r.FS)
}
