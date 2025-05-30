package duffel

import (
	"path"
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

func (p *Planner) CreateLink(pkgName, item string) {
	task := CreateLink{
		Action: "link",
		Path:   item,
		Dest:   path.Join(p.LinkPrefix, pkgName, item),
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
