package duffel

import (
	"encoding/json"
	"path/filepath"
)

func Execute(r *Request) error {
	linkPrefix, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return err
	}
	planner := NewPlanner(r.Target, linkPrefix)

	err = PlanInstallPackages(r.FS, planner, r.Source, r.Pkgs)

	plan := planner.Plan
	if r.DryRun {
		enc := json.NewEncoder(r.Stdout)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}
	return plan.Execute(r.FS)
}
