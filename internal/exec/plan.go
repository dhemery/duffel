package exec

import (
	"io/fs"
	"iter"
	"maps"
	"path"
	"slices"

	"github.com/dhemery/duffel/internal/analyze"
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
		item := name[targetLen:]
		task := NewTask(spec)
		if len(task) == 0 {
			continue
		}
		p.Tasks[item] = task
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

// NewTask returns a [Task] to bring some file to its planned state.
// The spec describes the current and planned states of the file.
func NewTask(spec analyze.Spec) Task {
	t := Task{}
	current, planned := spec.Current, spec.Planned
	if current.Equal(planned) {
		return t
	}

	switch {
	case current == nil:
	case current.Type == fs.ModeSymlink:
		t = append(t, Action{Action: ActRemove})
	}

	switch planned.Type {
	case fs.ModeDir:
		t = append(t, Action{Action: ActMkdir})
	case fs.ModeSymlink:
		t = append(t, Action{Action: "symlink", Dest: planned.Dest})
	default:
		panic("unknown planned mode: " + planned.Type.String())
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
