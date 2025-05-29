package duffel

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

func Execute(r *Request) error {
	linkPrefix, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}
	planner := &Planner{
		Target:     r.Target,
		Source:     r.Source,
		LinkPrefix: linkPrefix,
	}
	installer := &Installer{
		FS:      r.FS,
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
	return plan.Execute(r.FS, r.Target)
}
