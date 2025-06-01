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
	Plan           Plan
	Statuses       map[string]bool
}

func NewPlanner(target, targetToSource string) *Planner {
	return &Planner{
		TargetToSource: targetToSource,
		Plan:           Plan{Target: target},
		Statuses:       map[string]bool{},
	}
}

func (p *Planner) Exists(target string) bool {
	_, ok := p.Statuses[target]
	return ok
}

func (p *Planner) CreateLink(pkg, item string) {
	task := CreateLink{
		Action: "link",
		Item:   item,
		Dest:   path.Join(p.TargetToSource, pkg, item),
	}
	p.Statuses[item] = true
	p.Plan.Tasks = append(p.Plan.Tasks, task)
}

type CreateLink struct {
	Action string
	Item   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS, target string) error {
	return fsys.Symlink(a.Dest, path.Join(target, a.Item))
}
