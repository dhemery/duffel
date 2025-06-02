package duffel

import (
	"encoding/json"
)

func Execute(r *Request) error {
	planner := NewPlanner(r.Target)

	err := PlanInstallPackages(r.FS, planner, r.Target, r.Source, r.Pkgs)

	tasks := planner.Tasks()
	plan := Plan{Target: r.Target, Tasks: tasks}
	if r.DryRun {
		enc := json.NewEncoder(r.Stdout)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}
	return plan.Execute(r.FS)
}
