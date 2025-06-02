package duffel

import (
	"encoding/json"
	"io"
)

func Execute(r *Request, dryRun bool, w io.Writer) error {
	planner := Planner{}

	err := PlanInstallPackages(r.FS, planner, r.Target, r.Source, r.Pkgs)

	tasks := planner.Tasks()
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
