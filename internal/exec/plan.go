package exec

import (
	"io/fs"
	"iter"
	"maps"
	"path"
	"slices"

	"github.com/dhemery/duffel/internal/analyze"
)

// A Plan is the sequence of tasks
// to bring the file tree rooted at Target to the desired state.
type Plan struct {
	Target string          `json:"target"`
	Tasks  map[string]Task `json:"tasks"`
}

type Specs interface {
	All() iter.Seq2[string, analyze.Spec]
}

func New(target string, specs Specs) Plan {
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

func NewTask(spec analyze.Spec) Task {
	t := Task{}
	current, planned := spec.Current, spec.Planned
	if current.Equal(planned) {
		return t
	}

	switch {
	case current == nil:
	case current.Type == fs.ModeSymlink:
		t = append(t, FileOp{Op: OpRemove})
	}

	switch planned.Type {
	case fs.ModeDir:
		t = append(t, FileOp{Op: OpMkdir})
	case fs.ModeSymlink:
		t = append(t, FileOp{Op: "symlink", Dest: planned.Dest})
	default:
		panic("unknown planned mode: " + planned.Type.String())
	}

	return t
}

// A Task describes the work to bring a file in the target tree to a desired state.
type Task []FileOp

func (t Task) Execute(fsys fs.FS, name string) error {
	for _, op := range t {
		if err := op.Execute(fsys, name); err != nil {
			return err
		}
	}
	return nil
}
