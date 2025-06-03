package duffel

import (
	"encoding/json"
	"io"
)

func Execute(r *Request, dryRun bool, w io.Writer) error {
	image := Image{}

	err := PlanInstallPackages(r.FS, r.Source, r.Target, r.Pkgs, image)

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
