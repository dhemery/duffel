package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type Conflict struct{}

func (e *Conflict) Error() string {
	return ""
}

func PlanInstallPackages(fsys FS, planner *Planner, source string, pkgs []string) error {
	for _, pkg := range pkgs {
		pkgDir := path.Join(source, pkg)
		err := fs.WalkDir(fsys, pkgDir, PlanInstallPackage(planner, pkgDir, pkg))
		if err != nil {
			return err
		}
	}

	return nil
}

func PlanInstallPackage(planner *Planner, pkgDir string, pkg string) fs.WalkDirFunc {
	return func(dir string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't install the pkg dir itself
		if dir == pkgDir {
			return nil
		}

		itemPath, _ := filepath.Rel(pkgDir, dir)
		if planner.Exists(itemPath) {
			return &Conflict{}
		}
		planner.CreateLink(pkg, itemPath)

		return nil
	}
}
