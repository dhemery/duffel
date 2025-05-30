package duffel

import (
	"path"
	"path/filepath"
)

type Task interface {
	Execute(fsys FS, target string) error
}

type Plan struct {
	Target string
	Tasks  []Task
}

func (p *Plan) Execute(fsys FS) error {
	for _, task := range p.Tasks {
		if err := task.Execute(fsys, p.Target); err != nil {
			return err
		}
	}
	return nil
}

type Planner struct {
	LinkPrefix string
	Plan       Plan
}

func NewPlanner(source, target string) (*Planner, error) {
	linkPrefix, err := filepath.Rel(target, source)
	if err != nil {
		return nil, err
	}
	p := &Planner{
		LinkPrefix: linkPrefix,
	}
	p.Plan.Target = target
	return p, nil
}

func (p *Planner) CreateLink(pkgName, item string) {
	task := CreateLink{
		Action: "link",
		Path:   item,
		Dest:   path.Join(p.LinkPrefix, pkgName, item),
	}
	p.Plan.Tasks = append(p.Plan.Tasks, task)
}

type CreateLink struct {
	Action string
	Path   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS, target string) error {
	return fsys.Symlink(a.Dest, path.Join(target, a.Path))
}
