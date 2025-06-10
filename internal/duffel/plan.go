package duffel

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
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
func NewPlan(target string, targetGap TargetGap) Plan {
	tasks := make([]Task, 0)
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(targetGap)) {
		fileGap := targetGap[item]
		if fileGap.Desired == nil {
			continue
		}

		task := Task{Item: item, FileState: *fileGap.Desired}
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
	// FileState describes the desired state of the file.
	FileState
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
}

// MarshalJSON returns the JSON representation of t.
// It overrides the [FileState.MarshalJSON] promoted from the embedded FileState,
// which marshals only the FileState fields.
func (t Task) MarshalJSON() ([]byte, error) {
	stateJSON, err := t.FileState.MarshalJSON()
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

// A TargetGap describes the difference between the current and desired
// files in the target tree.
type TargetGap map[string]FileGap

// A FileGap describes the difference between the current and desired states
// of a file.
type FileGap struct {
	Current *FileState `json:"current,omitzero"`
	Desired *FileState `json:"desired,omitzero"`
}

// NewFileGap returns a FileGap with both the current and desired states
// set to the given mode and dest.
func NewFileGap(mode fs.FileMode, dest string) FileGap {
	return FileGap{
		Current: &FileState{Mode: mode, Dest: dest},
		Desired: &FileState{Mode: mode, Dest: dest},
	}
}

func (g FileGap) String() string {
	return fmt.Sprintf("%T{Current:%v,Desired:%v}", g, g.Current, g.Desired)
}

// A FileState describes the current or desired state of a file.
type FileState struct {
	Mode fs.FileMode
	Dest string
}

// MarshalJSON returns the JSON representation of s.
// It represents the Mode field as a descriptive string
// by calling [fs.FileMode.String] on the Mode.
func (s FileState) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Mode string `json:"mode"`
		Dest string `json:"dest,omitzero"`
	}{
		Mode: s.Mode.String(),
		Dest: s.Dest,
	})
}
