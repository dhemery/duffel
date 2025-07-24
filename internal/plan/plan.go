package plan

import (
	"io/fs"
	"path"
)

// A Plan is the sequence of tasks
// to bring the file tree rooted at Target to the desired state.
type Plan struct {
	Target string `json:"target"`
	Tasks  []Task `json:"tasks"`
}

func New(target string, specs Specs) Plan {
	p := Plan{Target: target, Tasks: []Task{}}
	for item, spec := range specs.All() {
		item := item[len(target)+1:]
		task := NewTask(item, spec)
		p.Tasks = append(p.Tasks, task)
	}
	return p
}

func (p Plan) Execute(fsys fs.FS) error {
	for _, task := range p.Tasks {
		if err := task.Execute(fsys, p.Target); err != nil {
			return err
		}
	}
	return nil
}

type FileOp interface {
	Execute(fsys fs.FS, target string) error
}

func NewTask(item string, spec Spec) Task {
	t := Task{Item: item}
	current, planned := spec.Current, spec.Planned
	if current.Equal(planned) {
		return t
	}

	switch {
	case current == nil:
	case current.Type == fs.ModeSymlink:
		t.Ops = append(t.Ops, RemoveOp)
	}

	switch planned.Type {
	case fs.ModeDir:
		t.Ops = append(t.Ops, MkDirOp)
	case fs.ModeSymlink:
		t.Ops = append(t.Ops, NewSymlinkOp(planned.Dest))
	}

	return t
}

// A Task describes the work to bring a file in the target tree to a desired state.
type Task struct {
	Item string   // Item is the path of the file relative to target.
	Ops  []FileOp // The file operations to bring the item to the desired state.
}

func (t Task) Execute(fsys fs.FS, target string) error {
	name := path.Join(target, t.Item)
	for _, op := range t.Ops {
		if err := op.Execute(fsys, name); err != nil {
			return err
		}
	}
	return nil
}
