package plan

import (
	"iter"
	"maps"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type specMap map[string]spec

func (sm specMap) all() iter.Seq2[string, spec] {
	return maps.All(sm)
}

func TestNewPlan(t *testing.T) {
	tests := map[string]struct {
		specs     specMap
		wantTasks map[string]Task
	}{
		"plans no task if current and planned are equal": {
			specs: specMap{
				// Each spec's planned state is the same as the current state,
				// so the target tree is already at the planned state.
				"target/no-file": spec{
					current: file.NoFileState(),
					planned: file.NoFileState(),
				},
				"target/dir": spec{
					current: file.DirState(),
					planned: file.DirState(),
				},
				"target/file": spec{
					current: file.FileState(),
					planned: file.FileState(),
				},
				"target/link/same/dest/file": spec{
					current: file.LinkState("some/dest", file.TypeFile),
					planned: file.LinkState("some/dest", file.TypeFile),
				},
				"target/link/same/dest/dir": spec{
					current: file.LinkState("some/dest", file.TypeDir),
					planned: file.LinkState("some/dest", file.TypeDir),
				},
				"target/link/same/dest/symlink": spec{
					current: file.LinkState("some/dest", file.TypeSymlink),
					planned: file.LinkState("some/dest", file.TypeSymlink),
				},
			},
			wantTasks: map[string]Task{},
		},
		"plans tasks if current and planned differ": {
			specs: specMap{
				"target/new-dir": spec{
					current: file.NoFileState(),
					planned: file.DirState(),
				},
				"target/new-link": spec{
					current: file.NoFileState(),
					planned: file.LinkState("some/dest", file.TypeFile),
				},
				"target/link-to-dir": spec{
					current: file.LinkState("some/dest", file.TypeFile),
					planned: file.DirState(),
				},
			},
			wantTasks: map[string]Task{
				"link-to-dir": {file.RemoveAction(), file.MkdirAction()},
				"new-dir":     {file.MkdirAction()},
				"new-link":    {file.SymlinkAction("some/dest")},
			},
		},
	}

	for desc, test := range tests {
		const target = "target"
		t.Run(desc, func(t *testing.T) {
			p := newPlan(target, test.specs)

			wantPlan := Plan{Target: target, Tasks: test.wantTasks}

			if diff := cmp.Diff(wantPlan, p); diff != "" {
				t.Error("Plan: ", diff)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	tests := map[string]struct {
		current  file.State
		planned  file.State
		wantTask Task
	}{
		"from no file to symlink": {
			current:  file.NoFileState(),
			planned:  file.LinkState("../planned/dest", file.TypeFile),
			wantTask: Task{{Action: "symlink", Dest: "../planned/dest"}},
		},
		"from no file to dir": {
			current:  file.NoFileState(),
			planned:  file.DirState(),
			wantTask: Task{file.MkdirAction()},
		},
		"from symlink to dir": {
			current:  file.LinkState("some/dest", file.TypeFile),
			planned:  file.DirState(),
			wantTask: Task{file.RemoveAction(), file.MkdirAction()},
		},
	}

	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			gotTask := newTask(test.current, test.planned)

			wantTask := test.wantTask

			if diff := cmp.Diff(wantTask, gotTask, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("NewTask():\n%s", diff)
			}
		})
	}
}
