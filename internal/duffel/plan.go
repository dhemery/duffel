package duffel

import (
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

// A Plan is the sequence of tasks
// to bring the file tree rooted at Target to the desired state.
type Plan struct {
	Target string `json:"target"`
	Tasks  []Task `json:"tasks"`
}

// NewPlan returns a Plan with tasks to bring
// a set of files to their desired states.
// Target is the root directory for the set of files.
// TargetGap describes the current and desired
// state of each file to include in the plan.
func NewPlan(target string, targetGap Index) Plan {
	tasks := make([]Task, 0)
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(targetGap)) {
		fileGap := targetGap[item]
		if fileGap.Desired == nil {
			continue
		}

		task := Task{Item: item, State: *fileGap.Desired}
		tasks = append(tasks, task)
	}
	return Plan{
		Target: target,
		Tasks:  tasks,
	}
}

func (p Plan) Execute(fsys FS) error {
	for _, task := range p.Tasks {
		if err := task.Execute(fsys, p.Target); err != nil {
			return err
		}
	}
	return nil
}

// A Task describes the work to bring a file in the target tree to a desired state.
type Task struct {
	// Item is the path of the file relative to target.
	Item string
	// State describes the desired state of the file.
	file.State
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
}

// MarshalJSON returns the JSON representation of t.
// It overrides the [file.State.MarshalJSON] promoted from the embedded State,
// which marshals only the State fields.
func (t Task) MarshalJSON() ([]byte, error) {
	stateJSON, err := t.State.MarshalJSON()
	if err != nil {
		return nil, err
	}
	taskJSON, err := json.Marshal(struct {
		Item string `json:"item"`
	}{
		Item: t.Item,
	})
	if err != nil {
		return nil, err
	}
	// Replace the closing brace with a comma to continue with the state
	taskJSON[len(taskJSON)-1] = ','
	// Skip the the state's opening brace to continue after the task fields
	stateJSON = stateJSON[1:]
	return append(taskJSON, stateJSON...), nil
}

// An Index collects Specs by item name.
type Index map[string]Spec

// A Spec describes the current and desired states of a target item file.
type Spec struct {
	Current *file.State `json:"current,omitzero"`
	Desired *file.State `json:"desired,omitzero"`
}

func (s Spec) String() string {
	return fmt.Sprintf("%T{Current:%v,Desired:%v}", s, s.Current, s.Desired)
}
