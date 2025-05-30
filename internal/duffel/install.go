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
		err := fs.WalkDir(i.FS, pkgDir, func(dir string, d fs.DirEntry, err error) error {
			if err != nil || dir == pkgDir {
				return err
			}
			itemPath, _ := filepath.Rel(pkgDir, dir)
			i.Planner.CreateLink(pkg, itemPath)

			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
