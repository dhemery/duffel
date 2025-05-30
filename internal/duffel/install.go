package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type Installer struct {
	FS      FS
	Source  string
	Planner *Planner
}

func (i *Installer) Plan(pkgs []string) error {
	for _, pkg := range pkgs {
		pkgDir := path.Join(i.Source, pkg)
		err := fs.WalkDir(i.FS, pkgDir, PlanInstallPackage(pkgDir, pkg, i.Planner))
		if err != nil {
			return err
		}
	}
	return nil
}

func PlanInstallPackage(pkgDir string, pkg string, planner *Planner) fs.WalkDirFunc {
	return func(dir string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't install the pkg dir itself
		if dir == pkgDir {
			return nil
		}

		itemPath, _ := filepath.Rel(pkgDir, dir)
		planner.CreateLink(pkg, itemPath)

		return nil
	}
}
