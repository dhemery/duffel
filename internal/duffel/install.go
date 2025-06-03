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
	v := InstallPlanner{
		target:         r.Target,
		targetToSource: targetToSource,
	}
	sourcePkg := path.Join(r.Source, pkg)
	return func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't visit sourcePkg
		if path == sourcePkg {
			return nil
		}

		item, _ := filepath.Rel(sourcePkg, path)

		return v.Visit(r.Source, pkg, item, image)
	}
}

type ItemVisitor interface {
	Visit(source, pkg, item string, image Image) error
}

type InstallPlanner struct {
	target         string
	targetToSource string
}

func (v InstallPlanner) Visit(source, pkg, item string, image Image) error {
	status := image.Status(item)
	if status.WillExist() {
		return &ErrConflict{}
	}

	dest := path.Join(v.targetToSource, pkg, item)
	state := State{Dest: dest}
	image.Create(item, state)

	return nil
}
