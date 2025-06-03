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

type ItemVisitor interface {
	Visit(source, pkg, item string, image Image) error
}

type PkgPlanner struct {
	FS      fs.FS
	Source  string
	Pkg     string
	Visitor ItemVisitor
}

func (p PkgPlanner) Plan(image Image) error {
	sourcePkg := path.Join(p.Source, p.Pkg)
	return fs.WalkDir(p.FS, sourcePkg, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't visit sourcePkg
		if path == sourcePkg {
			return nil
		}

		item, _ := filepath.Rel(sourcePkg, path)

		return p.Visitor.Visit(p.Source, p.Pkg, item, image)
	})
}

func PlanInstallPackages(fsys fs.FS, source string, target string, pkgs []string, image Image) error {
	targetToSource, err := filepath.Rel(target, source)
	if err != nil {
		return err
	}
	install := InstallVisitor{
		target:         target,
		targetToSource: targetToSource,
	}

	var planners []PkgPlanner
	for _, pkg := range pkgs {
		planner := PkgPlanner{
			FS:      fsys,
			Source:  source,
			Pkg:     pkg,
			Visitor: install,
		}
		planners = append(planners, planner)
	}

	for _, planner := range planners {
		err := planner.Plan(image)
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
