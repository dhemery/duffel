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

func PlanInstallPackages(fsys FS, planner *Planner, target string, source string, pkgs []string) error {
	targetToSource, err := filepath.Rel(target, source)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		sourcePkg := path.Join(source, pkg)
		err := fs.WalkDir(fsys, sourcePkg, PlanInstallPackage(planner, targetToSource, sourcePkg, pkg))
		if err != nil {
			return err
		}
	}

	return nil
}

func PlanInstallPackage(planner *Planner, targetToSource string, sourcePkg string, pkg string) fs.WalkDirFunc {
	return func(sourcePkgItem string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't install the pkg dir itself
		if sourcePkgItem == sourcePkg {
			return nil
		}

		item, _ := filepath.Rel(sourcePkg, sourcePkgItem)
		status := planner.Status(item)
		if status != nil {
			return &ErrConflict{}
		}

		dest := path.Join(targetToSource, pkg, item)
		result := &Result{Dest: dest}
		planner.Create(item, result)

		return nil
	}
}
