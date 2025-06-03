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

func PlanInstallPackages(r *Request, image Image) error {
	targetToSource, err := filepath.Rel(r.Target, r.Source)
	if err != nil {
		return err
	}
	for _, pkg := range r.Pkgs {
		sourcePkg := path.Join(r.Source, pkg)
		err := fs.WalkDir(r.FS, sourcePkg, PlanInstallPackage(r, image, targetToSource, pkg))
		if err != nil {
			return err
		}
	}

	return nil
}

func PlanInstallPackage(r *Request, image Image, targetToSource string, pkg string) fs.WalkDirFunc {
	sourcePkg := path.Join(r.Source, pkg)
	return func(sourcePkgItem string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't install the pkg dir itself
		if sourcePkgItem == sourcePkg {
			return nil
		}

		item, _ := filepath.Rel(sourcePkg, sourcePkgItem)

		status := image.Status(item)
		if status.WillExist() {
			return &ErrConflict{}
		}

		dest := path.Join(targetToSource, pkg, item)
		state := State{Dest: dest}
		image.Create(item, state)

		return nil
	}
}
