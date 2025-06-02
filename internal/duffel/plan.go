package duffel

import (
	"maps"
	"path"
	"slices"
)

type Result struct {
	Dest string `json:"dest"`
}

func (r Result) Exists() bool {
	return r.Dest != ""
}

type Task struct {
	// Item is the path of the item to create, relative to target
	Item string `json:"item"`

	// Result is the result to create at the target item path
	Result
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
}

type Status struct {
	Prior   Result
	Planned Result
}

func (s Status) WillExist() bool {
	return s.Prior.Exists() || s.Planned.Exists()
}

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

type Planner map[string]Status

func (p Planner) Create(item string, result Result) {
	p[item] = Status{Planned: result}
}

func (p Planner) Status(item string) Status {
	return p[item]
}

func (p Planner) Tasks() []Task {
	var tasks []Task
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(p)) {
		status := p[item]
		if !status.Planned.Exists() {
			continue
		}

		task := Task{Item: item, Result: status.Planned}
		tasks = append(tasks, task)
	}
	return tasks
}
