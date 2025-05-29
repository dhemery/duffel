package duffel

import (
	"fmt"
	"path"
)

type Installer struct {
	FS      FS
	Planner *Planner
}

func (i *Installer) PlanPackages(pkgs []string) error {
	for _, pkg := range pkgs {
		if err := i.PlanPackage(pkg); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) PlanPackage(pkg string) error {
	pkgDir := path.Join(i.Planner.Source, pkg)
	entries, err := i.FS.ReadDir(pkgDir)
	if err != nil {
		return fmt.Errorf("reading package dir %s: %w", pkgDir, err)
	}
	for _, e := range entries {
		i.Planner.CreateLink(pkgDir, e.Name())
	}
	return nil
}
