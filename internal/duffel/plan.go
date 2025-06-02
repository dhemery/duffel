package duffel

import (
	"path"
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
	TargetToSource string
	tasks          []Task
	Statuses       map[string]bool
}

func NewPlanner(target, targetToSource string) *Planner {
	return &Planner{
		TargetToSource: targetToSource,
		Statuses:       map[string]bool{},
	}
}

func (p *Planner) Tasks() []Task {
	return p.tasks
}

func (p *Planner) Exists(target string) bool {
	exists, ok := p.Statuses[target]
	return ok && exists
}

func (p *Planner) CreateLink(pkg, item string) {
	task := CreateLink{
		Action: "link",
		Item:   item,
		Dest:   path.Join(p.TargetToSource, pkg, item),
	}
	p.Statuses[item] = true
	p.tasks = append(p.tasks, task)
}

type CreateLink struct {
	Action string
	Item   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS, target string) error {
	return fsys.Symlink(a.Dest, path.Join(target, a.Item))
}
