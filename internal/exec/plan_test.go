package exec_test

import (
	"io/fs"
	"iter"
	"maps"
	"testing"

	. "github.com/dhemery/duffel/internal/exec"

	"github.com/dhemery/duffel/internal/analyze"
	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type specMap map[string]analyze.Spec

func (sm specMap) All() iter.Seq2[string, analyze.Spec] {
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
				"target/nil": analyze.Spec{
					Current: nil,
					Planned: nil,
				},
				"target/dir": analyze.Spec{
					Current: file.DirState(),
					Planned: file.DirState(),
				},
				"target/file": analyze.Spec{
					Current: file.FileState(),
					Planned: file.FileState(),
				},
				"target/link/same/dest/file": analyze.Spec{
					Current: file.LinkState("some/dest", 0),
					Planned: file.LinkState("some/dest", 0),
				},
				"target/link/same/dest/dir": analyze.Spec{
					Current: file.LinkState("some/dest", fs.ModeDir),
					Planned: file.LinkState("some/dest", fs.ModeDir),
				},
				"target/link/same/dest/symlink": analyze.Spec{
					Current: file.LinkState("some/dest", fs.ModeSymlink),
					Planned: file.LinkState("some/dest", fs.ModeSymlink),
				},
			},
			wantTasks: map[string]Task{},
		},
		"plans tasks if current and planned differ": {
			specs: specMap{
				"target/new-dir": analyze.Spec{
					Current: nil,
					Planned: file.DirState(),
				},
				"target/new-link": analyze.Spec{
					Current: nil,
					Planned: file.LinkState("some/dest", 0),
				},
				"target/link-to-dir": analyze.Spec{
					Current: file.LinkState("some/dest", 0),
					Planned: file.DirState(),
				},
			},
			wantTasks: map[string]Task{
				"link-to-dir": {{Action: ActRemove}, {Action: ActMkdir}},
				"new-dir":     {{Action: ActMkdir}},
				"new-link":    {{Action: "symlink", Dest: "some/dest"}},
			},
		},
	}

	for desc, test := range tests {
		const target = "target"
		t.Run(desc, func(t *testing.T) {
			plan := NewPlan(target, test.specs)

			wantPlan := Plan{Target: target, Tasks: test.wantTasks}

			if diff := cmp.Diff(wantPlan, plan); diff != "" {
				t.Error("Plan: ", diff)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	tests := map[string]struct {
		current  *file.State
		planned  *file.State
		wantTask Task
	}{
		"from nil to symlink": {
			current:  nil,
			planned:  file.LinkState("../planned/dest", 0),
			wantTask: Task{{Action: "symlink", Dest: "../planned/dest"}},
		},
		"from nil to dir": {
			current:  nil,
			planned:  file.DirState(),
			wantTask: Task{{Action: ActMkdir}},
		},
		"from symlink to dir": {
			current:  file.LinkState("some/dest", 0),
			planned:  file.DirState(),
			wantTask: Task{{Action: ActRemove}, {Action: ActMkdir}},
		},
	}

	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			gotTask := NewTask(test.current, test.planned)

			wantTask := test.wantTask

			if diff := cmp.Diff(wantTask, gotTask, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("NewTask():\n%s", diff)
			}
		})
	}
}
