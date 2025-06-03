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
	install := InstallVisitor{
		source:         r.Source,
		target:         r.Target,
		targetToSource: targetToSource,
		image:          image,
	}

	var planners []PkgPlanner
	for _, pkg := range r.Pkgs {
		planner := NewPkgPlanner(r.FS, r.Source, pkg, install)
		planners = append(planners, planner)
	}

	for _, planner := range planners {
		err = planner.Plan()
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
