package duffel

import (
	"encoding/json"
)

func Execute(r *Request) error {
	planner, err := NewPlanner(r.Source, r.Target)
	if err != nil {
		return err
	}
	installer := &Installer{
		FS:      r.FS,
		Source:  r.Source,
		Planner: planner,
	}
	err = installer.PlanPackages(r.Pkgs)
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
