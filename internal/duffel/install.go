package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type Installer struct {
	FS      FS
	Planner *Planner
}

func (i *Installer) PlanPackages(pkgs []string) error {
	for _, pkg := range pkgs {
		pkgDir := path.Join(i.Planner.Source, pkg)
		err := fs.WalkDir(i.FS, pkgDir, func(dir string, d fs.DirEntry, err error) error {
			if d == nil {
				return err
			}
			if dir == pkgDir {
				return nil
			}
			itemPath, _ := filepath.Rel(pkgDir, dir)
			i.Planner.CreateLink(pkg, itemPath)

			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
