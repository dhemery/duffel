package duffel

import (
	"io/fs"
	"path"
	"path/filepath"
)

type Plan struct {
	Target string `json:"target"`
	Tasks  []Task `json:"tasks"`
}

func (p *Plan) Execute(fsys FS) error {
	for _, task := range p.Tasks {
		if err := task.Execute(fsys, p.Target); err != nil {
			return err
		}
	}
	return nil
}

type Task struct {
	// Item is the path of the item to create, relative to target
	Item string `json:"item"`

	// State describes the file to create at the target item path
	State
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
}

func PlanInstallPackages(fsys fs.FS, source string, target string, pkgs []string, image Image) error {
	targetToSource, err := filepath.Rel(target, source)
	install := InstallVisitor{
		target:         target,
		targetToSource: targetToSource,
	}
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		sourcePkg := path.Join(source, pkg)
		err := fs.WalkDir(fsys, sourcePkg, PlanInstallPackage(source, pkg, install, image))
		if err != nil {
			return err
		}
	}

	return nil
}

func PlanInstallPackage(source string, pkg string, v ItemVisitor, image Image) fs.WalkDirFunc {
	sourcePkg := path.Join(source, pkg)
	return func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't visit sourcePkg
		if path == sourcePkg {
			return nil
		}

		item, _ := filepath.Rel(sourcePkg, path)

		return v.Visit(source, pkg, item, image)
	}
}
