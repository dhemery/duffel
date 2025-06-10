package duffel

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
)

type Plan struct {
	Target string `json:"target"`
	Tasks  []Task `json:"tasks"`
}

// NewPlan returns a Plan with tasks to close the gap in the tree rooted at target.
func NewPlan(target string, gap map[string]Status) Plan {
	tasks := make([]Task, 0)
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(gap)) {
		status := gap[item]
		if status.Desired == nil {
			continue
		}

		task := Task{Item: item, State: *status.Desired}
		tasks = append(tasks, task)
	}
	return Plan{
		Target: target,
		Tasks:  tasks,
	}
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
	// Item is the path of the item to create, relative to target.
	Item string
	// State describes the file to create at the target item path.
	State
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
}

type TargetTree map[string]Status

func (tt TargetTree) Set(item string, status Status) {
	tt[item] = status
}

func (tt TargetTree) Status(item string) (Status, bool) {
	s, ok := tt[item]
	return s, ok
}

func (tt TargetTree) Tasks() []Task {
	var tasks []Task
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(tt)) {
		status := tt[item]
		if status.Desired == nil {
			continue
		}

		task := Task{Item: item, State: *status.Desired}
		tasks = append(tasks, task)
	}
	return tasks
}

func NewStatus(mode fs.FileMode, dest string) Status {
	return Status{
		Current: &State{Mode: mode, Dest: dest},
		Desired: &State{Mode: mode, Dest: dest},
	}
}

type Status struct {
	Current *State `json:"current,omitzero"`
	Desired *State `json:"desired,omitzero"`
}

func (s Status) String() string {
	return fmt.Sprintf("Status{Current:%v,Desired:%v}", s.Current, s.Desired)
}

type State struct {
	Mode fs.FileMode
	Dest string
}

// MarshalJSON implements [json.Marshaller].
// It makes the JSON more descriptive by calling [fs.FileMode.String] on the Mode.
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Mode string `json:"mode"`
		Dest string `json:"dest,omitzero"`
	}{
		Mode: s.Mode.String(),
		Dest: s.Dest,
	})
}
