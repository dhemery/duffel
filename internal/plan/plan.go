package plan

import (
	"encoding/json"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
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
		state := spec.Planned
		if state == nil {
			continue
		}
		relItem := item[len(target)+1:]
		task := Task{Item: relItem, State: *state}
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
	Item       string   // Item is the path of the file relative to target.
	Ops        []FileOp // The file operations to bring the item to the desired state.
	file.State          // State describes the desired state of the file.
}

func (t Task) Execute(fsys fs.FS, target string) error {
	return file.Symlink(fsys, t.Dest, path.Join(target, t.Item))
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
