package analyze_test

import (
	"iter"
	"maps"
	"testing"

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
		wantTasks map[string]analyze.Task
	}{
		"plans no task if current and planned are equal": {
			specs: specMap{
				// Each spec's planned state is the same as the current state,
				// so the target tree is already at the planned state.
				"target/no-file": analyze.Spec{
					Current: file.NoFileState(),
					Planned: file.NoFileState(),
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
					Current: file.LinkState("some/dest", file.TypeFile),
					Planned: file.LinkState("some/dest", file.TypeFile),
				},
				"target/link/same/dest/dir": analyze.Spec{
					Current: file.LinkState("some/dest", file.TypeDir),
					Planned: file.LinkState("some/dest", file.TypeDir),
				},
				"target/link/same/dest/symlink": analyze.Spec{
					Current: file.LinkState("some/dest", file.TypeSymlink),
					Planned: file.LinkState("some/dest", file.TypeSymlink),
				},
			},
			wantTasks: map[string]analyze.Task{},
		},
		"plans tasks if current and planned differ": {
			specs: specMap{
				"target/new-dir": analyze.Spec{
					Current: file.NoFileState(),
					Planned: file.DirState(),
				},
				"target/new-link": analyze.Spec{
					Current: file.NoFileState(),
					Planned: file.LinkState("some/dest", file.TypeFile),
				},
				"target/link-to-dir": analyze.Spec{
					Current: file.LinkState("some/dest", file.TypeFile),
					Planned: file.DirState(),
				},
			},
			wantTasks: map[string]analyze.Task{
				"link-to-dir": {{Action: analyze.ActRemove}, {Action: analyze.ActMkdir}},
				"new-dir":     {{Action: analyze.ActMkdir}},
				"new-link":    {{Action: "symlink", Dest: "some/dest"}},
			},
		},
	}

	for desc, test := range tests {
		const target = "target"
		t.Run(desc, func(t *testing.T) {
			plan := analyze.NewPlan(target, test.specs)

			wantPlan := analyze.Plan{Target: target, Tasks: test.wantTasks}

			if diff := cmp.Diff(wantPlan, plan); diff != "" {
				t.Error("Plan: ", diff)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	tests := map[string]struct {
		current  file.State
		planned  file.State
		wantTask analyze.Task
	}{
		"from no file to symlink": {
			current:  file.NoFileState(),
			planned:  file.LinkState("../planned/dest", file.TypeFile),
			wantTask: analyze.Task{{Action: "symlink", Dest: "../planned/dest"}},
		},
		"from no file to dir": {
			current:  file.NoFileState(),
			planned:  file.DirState(),
			wantTask: analyze.Task{{Action: analyze.ActMkdir}},
		},
		"from symlink to dir": {
			current:  file.LinkState("some/dest", file.TypeFile),
			planned:  file.DirState(),
			wantTask: analyze.Task{{Action: analyze.ActRemove}, {Action: analyze.ActMkdir}},
		},
	}

	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			gotTask := analyze.NewTask(test.current, test.planned)

			wantTask := test.wantTask

			if diff := cmp.Diff(wantTask, gotTask, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("NewTask():\n%s", diff)
			}
		})
	}
}
