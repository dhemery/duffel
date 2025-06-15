package plan

import (
	"encoding/json"
	"iter"
	"path"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/item"
)

type specList interface {
	ByItem() iter.Seq2[string, item.Spec]
}
type SymlinkFS interface {
	Symlink(oldname, newname string) error
}

// A Plan is the sequence of tasks
// to bring the file tree rooted at Target to the desired state.
type Plan struct {
	Target string `json:"target"`
	Tasks  []Task `json:"tasks"`
}

// New returns a Plan with tasks to bring
// a set of files to their desired states.
// Target is the root directory for the set of files.
// SpecList delivers the specs in the plan sorted by item name.
func New(target string, specs specList) Plan {
	tasks := make([]Task, 0)
	for item, spec := range specs.ByItem() {
		if spec.Desired == nil {
			continue
		}

		task := Task{Item: item, State: *spec.Desired}
		tasks = append(tasks, task)
	}
	return Plan{
		Target: target,
		Tasks:  tasks,
	}
}

func (p Plan) Execute(fsys SymlinkFS) error {
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

func (t Task) Execute(fsys SymlinkFS, target string) error {
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
