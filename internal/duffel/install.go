package duffel

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

func Install(r *Request) error {
	plan := Plan{}
	sourceLinkDest, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return fmt.Errorf("making source link dest: %w", err)
	}

	for _, pkg := range r.Pkgs {
		pkgDir := filepath.Join(r.Source, pkg)
		pkgLinkDest := filepath.Join(sourceLinkDest, pkg)
		entries, err := r.FS.ReadDir(pkgDir)
		if err != nil {
			return fmt.Errorf("reading package %s: %w", pkg, err)
		}
		for _, e := range entries {
			linkPath := filepath.Join(r.Target, e.Name())
			linkDest := filepath.Join(pkgLinkDest, e.Name())
			plan.CreateLink(linkPath, linkDest)
		}

	}
	if r.DryRun {
		enc := json.NewEncoder(r.Stdout)
		return enc.Encode(plan)
	}
	return plan.Execute(r.FS)
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
