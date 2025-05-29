package duffel

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
)

func Execute(r *Request) error {
	plan := &Plan{}
	sourceLinkDest, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}
	installer := &Installer{
		Request:        r,
		Plan:           plan,
		SourceLinkDest: sourceLinkDest,
	}
	err = installer.PlanPackages(r.Pkgs)
	if r.DryRun {
		enc := json.NewEncoder(r.Stdout)
		return enc.Encode(plan)
	}
	if err != nil {
		return err
	}
	return plan.Execute(r.FS)
}

type Installer struct {
	*Request
	Plan           *Plan
	SourceLinkDest string
}

func (i *Installer) PlanPackages(pkgs []string) error {
	for _, pkg := range pkgs {
		pkgDir := path.Join(i.Source, pkg)
		pkgLinkDest := path.Join(i.SourceLinkDest, pkg)
		if err := i.PlanPackage(pkgDir, pkgLinkDest); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) PlanPackage(pkgDir string, pkgLinkDest string) error {
	entries, err := i.FS.ReadDir(pkgDir)
	if err != nil {
		return fmt.Errorf("reading package %s: %w", pkgDir, err)
	}
	for _, e := range entries {
		linkPath := path.Join(i.Target, e.Name())
		linkDest := path.Join(pkgLinkDest, e.Name())
		i.Plan.CreateLink(linkPath, linkDest)
	}
	return nil
}

type Task interface {
	Execute(fsys FS) error
}

type Plan []Task

func (p *Plan) Execute(fsys FS) error {
	for _, task := range *p {
		if err := task.Execute(fsys); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plan) CreateLink(path, dest string) {
	(*p) = append(*p, CreateLink{Action: "link", Path: path, Dest: dest})
}

type CreateLink struct {
	Action string
	Path   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS) error {
	return fsys.Symlink(a.Dest, a.Path)
}
