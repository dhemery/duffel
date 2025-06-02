package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type ErrConflict struct{}

func (e *ErrConflict) Error() string {
	return ""
}

func PlanInstallPackages(fsys FS, planner *Planner, source string, pkgs []string) error {
	for _, pkg := range pkgs {
		sourcePkg := path.Join(source, pkg)
		err := fs.WalkDir(fsys, sourcePkg, PlanInstallPackage(planner, sourcePkg, pkg))
		if err != nil {
			return err
		}
	}

	return nil
}

func PlanInstallPackage(planner *Planner, sourcePkg string, pkg string) fs.WalkDirFunc {
	return func(sourcePkgItem string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't install the pkg dir itself
		if sourcePkgItem == sourcePkg {
			return nil
		}

		item, _ := filepath.Rel(sourcePkg, sourcePkgItem)
		if planner.Exists(item) {
			return &ErrConflict{}
		}
		planner.CreateLink(pkg, item)

		return nil
	}
}
