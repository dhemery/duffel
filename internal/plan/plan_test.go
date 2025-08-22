package plan_test

import (
	"iter"
	"maps"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/plan"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type specMap map[string]plan.Spec

func (sm specMap) All() iter.Seq2[string, plan.Spec] {
	return maps.All(sm)
}

func TestNewPlan(t *testing.T) {
	tests := map[string]struct {
		specs     specMap
		wantTasks map[string]plan.Task
	}{
		"plans no task if current and planned are equal": {
			specs: specMap{
				// Each spec's planned state is the same as the current state,
				// so the target tree is already at the planned state.
				"target/no-file": plan.Spec{
					Current: file.NoFileState(),
					Planned: file.NoFileState(),
				},
				"target/dir": plan.Spec{
					Current: file.DirState(),
					Planned: file.DirState(),
				},
				"target/file": plan.Spec{
					Current: file.FileState(),
					Planned: file.FileState(),
				},
				"target/link/same/dest/file": plan.Spec{
					Current: file.LinkState("some/dest", file.TypeFile),
					Planned: file.LinkState("some/dest", file.TypeFile),
				},
				"target/link/same/dest/dir": plan.Spec{
					Current: file.LinkState("some/dest", file.TypeDir),
					Planned: file.LinkState("some/dest", file.TypeDir),
				},
				"target/link/same/dest/symlink": plan.Spec{
					Current: file.LinkState("some/dest", file.TypeSymlink),
					Planned: file.LinkState("some/dest", file.TypeSymlink),
				},
			},
			wantTasks: map[string]plan.Task{},
		},
		"plans tasks if current and planned differ": {
			specs: specMap{
				"target/new-dir": plan.Spec{
					Current: file.NoFileState(),
					Planned: file.DirState(),
				},
				"target/new-link": plan.Spec{
					Current: file.NoFileState(),
					Planned: file.LinkState("some/dest", file.TypeFile),
				},
				"target/link-to-dir": plan.Spec{
					Current: file.LinkState("some/dest", file.TypeFile),
					Planned: file.DirState(),
				},
			},
			wantTasks: map[string]plan.Task{
				"link-to-dir": {file.RemoveAction(), file.MkdirAction()},
				"new-dir":     {file.MkdirAction()},
				"new-link":    {file.SymlinkAction("some/dest")},
			},
		},
	}

	for desc, test := range tests {
		const target = "target"
		t.Run(desc, func(t *testing.T) {
			p := plan.NewPlan(target, test.specs)

			wantPlan := plan.Plan{Target: target, Tasks: test.wantTasks}

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
		wantTask plan.Task
	}{
		"from no file to symlink": {
			current:  file.NoFileState(),
			planned:  file.LinkState("../planned/dest", file.TypeFile),
			wantTask: plan.Task{{Action: "symlink", Dest: "../planned/dest"}},
		},
		"from no file to dir": {
			current:  file.NoFileState(),
			planned:  file.DirState(),
			wantTask: plan.Task{file.MkdirAction()},
		},
		"from symlink to dir": {
			current:  file.LinkState("some/dest", file.TypeFile),
			planned:  file.DirState(),
			wantTask: plan.Task{file.RemoveAction(), file.MkdirAction()},
		},
	}

	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			gotTask := plan.NewTask(test.current, test.planned)

			wantTask := test.wantTask

			if diff := cmp.Diff(wantTask, gotTask, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("NewTask():\n%s", diff)
			}
		})
	}
}
