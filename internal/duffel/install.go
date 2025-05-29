package duffel

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
)

func Execute(r *Request) error {
	sourceLinkDest, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}
	planner := &Planner{
		Target:     r.Target,
		Source:     r.Source,
		LinkPrefix: sourceLinkDest,
		Plan:       Plan{},
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
	return plan.Execute(r.FS, r.Target)
}

type Installer struct {
	FS      FS
	Source  string
	Planner *Planner
}

func (i *Installer) PlanPackages(pkgs []string) error {
	for _, pkg := range pkgs {
		pkgDir := path.Join(i.Source, pkg)
		if err := i.PlanPackage(pkgDir); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) PlanPackage(pkgDir string) error {
	entries, err := i.FS.ReadDir(pkgDir)
	if err != nil {
		return fmt.Errorf("reading package %s: %w", pkgDir, err)
	}
	for _, e := range entries {
		i.Planner.CreateLink(pkgDir, e.Name())
	}
	return nil
}

type Task interface {
	Execute(fsys FS, target string) error
}

type Plan []Task

func (p *Plan) Execute(fsys FS, target string) error {
	for _, task := range *p {
		if err := task.Execute(fsys, target); err != nil {
			return err
		}
	}
	return nil
}

type Planner struct {
	Target     string
	Source     string
	LinkPrefix string
	Plan       Plan
}

func (p *Planner) CreateLink(pkgDir, item string) {
	pkgSourcePath, _ := filepath.Rel(p.Source, pkgDir)
	itemSourcePath := path.Join(pkgSourcePath, item)
	destPath := path.Join(p.LinkPrefix, itemSourcePath)
	task := CreateLink{
		Action: "link",
		Path:   item,
		Dest:   destPath,
	}
	p.Plan = append(p.Plan, task)
}

type CreateLink struct {
	Action string
	Path   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS, target string) error {
	return fsys.Symlink(a.Dest, path.Join(target, a.Path))
}
