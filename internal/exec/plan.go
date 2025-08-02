package exec

import (
	"io/fs"
	"iter"
	"maps"
	"path"
	"slices"

	"github.com/dhemery/duffel/internal/analyze"
	"github.com/dhemery/duffel/internal/file"
)

// A Plan is a set of tasks
// to bring the file tree rooted at Target to the desired state.
type Plan struct {
	Target string          `json:"target"`
	Tasks  map[string]Task `json:"tasks"`
}

type Specs interface {
	// All returns an iterator over the item name and [analyze.Spec]
	// for each file in the target tree that is not in its planned state.
	All() iter.Seq2[string, analyze.Spec]
}

// NewPlan returns a [Plan] to bring the les the target tree
// to its planned state.
// Specs describes the desired and planned states for each file
// that is not in its planned state.
func NewPlan(target string, specs Specs) Plan {
	targetLen := len(target) + 1
	p := Plan{Target: target, Tasks: map[string]Task{}}
	for name, spec := range specs.All() {
		if spec.Current == spec.Planned {
			continue
		}
		item := name[targetLen:]
		p.Tasks[item] = NewTask(spec.Current, spec.Planned)
	}
	return p
}

// Execute executes p's tasks against the target file tree.
func (p Plan) Execute(fsys fs.FS) error {
	for _, item := range slices.Sorted(maps.Keys(p.Tasks)) {
		task := p.Tasks[item]
		name := path.Join(p.Target, item)
		if err := task.Execute(fsys, name); err != nil {
			return err
		}
	}
	return nil
}

// NewTask creates a [Task] with the actions to bring file
// from the current state to the planned state.
func NewTask(current, planned file.State) Task {
	t := Task{}

	switch {
	case current.Type.IsNoFile(): // No-op
	case current.Type.IsLink():
		t = append(t, Action{Action: ActRemove})
	default:
		panic("do not know an action to remove " + current.Type.String())
	}

	switch {
	case planned.Type.IsNoFile(): // No-op
	case planned.Type.IsDir():
		t = append(t, Action{Action: ActMkdir})
	case planned.Type.IsLink():
		t = append(t, Action{Action: "symlink", Dest: planned.Dest})
	default:
		panic("do not know an action to create " + planned.Type.String())
	}

	return t
}

// A Task is a sequence of actions to bring a file to a desired state.
type Task []Action

// Execute executes t's actions on the named file.
func (t Task) Execute(fsys fs.FS, name string) error {
	for _, op := range t {
		if err := op.Execute(fsys, name); err != nil {
			return err
		}
	}
	return nil
}
