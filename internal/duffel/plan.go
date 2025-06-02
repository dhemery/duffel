package duffel

import (
	"path"
)

type Task interface {
	Execute(fsys FS, target string) error
}

type Result struct {
	Dest string
}

type Status struct {
	Prior   *Result
	Planned *Result
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
	status         map[string]*Status
}

func NewPlanner(target, targetToSource string) *Planner {
	return &Planner{
		TargetToSource: targetToSource,
		status:         map[string]*Status{},
	}
}

func (p *Planner) Create(item string, result *Result) {
	p.status[item] = &Status{Planned: result}
	task := CreateLink{
		Action: "link",
		Item:   item,
		Dest:   result.Dest,
	}
	p.tasks = append(p.tasks, task)
}

func (p *Planner) Status(item string) *Status {
	return p.status[item]
}

func (p *Planner) Tasks() []Task {
	return p.tasks
}

type CreateLink struct {
	Action string
	Item   string
	Dest   string
}

func (a CreateLink) Execute(fsys FS, target string) error {
	return fsys.Symlink(a.Dest, path.Join(target, a.Item))
}
