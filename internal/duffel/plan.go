package duffel

import (
	"path"
	"path/filepath"
)

type Task interface {
	Execute(fsys FS, target string) error
}

type Plan []Task

func (p *Plan) Execute(fsys FS, target string) error {
	for _, task := range *p {
		if err := task.Execute(fsys, target); err != nil {
			return err
		}
	}
	return nil
}

type Planner struct {
	Target     string
	Source     string
	LinkPrefix string
	Plan       Plan
}

func (p *Planner) CreateLink(pkgDir, item string) {
	pkgSourcePath, _ := filepath.Rel(p.Source, pkgDir)
	itemSourcePath := path.Join(pkgSourcePath, item)
	destPath := path.Join(p.LinkPrefix, itemSourcePath)
	task := CreateLink{
		Action: "link",
		Path:   item,
		Dest:   destPath,
	}
	p.Plan = append(p.Plan, task)
}

type CreateLink struct {
	Action string
	Path   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS, target string) error {
	return fsys.Symlink(a.Dest, path.Join(target, a.Path))
}
