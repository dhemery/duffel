package plan

import (
	"io/fs"
	"iter"
	"maps"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type specMap map[string]Spec

func (sm specMap) All() iter.Seq2[string, Spec] {
	return maps.All(sm)
}

func TestNewPlan(t *testing.T) {
	tests := map[string]struct {
		specs     specMap
		wantTasks []Task
	}{
		"omit do nothing tasks": {
			specs: specMap{
				// Each spec's planned state is the same as the current state,
				// so the target tree is already at the planned state.
				"target/dir": Spec{
					Current: dirState(),
					Planned: dirState(),
				},
				"target/file": Spec{
					Current: fileState(),
					Planned: fileState(),
				},
				"target/link": Spec{
					Current: linkState("some/dest", 0),
					Planned: linkState("some/dest", 0),
				},
			},
			wantTasks: []Task{},
		},
		"several tasks": {
			specs: specMap{
				"target/new-dir": Spec{
					Current: nil,
					Planned: dirState(),
				},
				"target/new-link": Spec{
					Current: nil,
					Planned: linkState("some/dest", 0),
				},
				"target/link-to-dir": Spec{
					Current: linkState("some/dest", 0),
					Planned: dirState(),
				},
			},
			wantTasks: []Task{ // Note: Sorted by item.
				{
					Item: "link-to-dir",
					Ops: []FileOp{
						RemoveOp,
						MkDirOp,
					},
				},
				{
					Item: "new-dir",
					Ops: []FileOp{
						MkDirOp,
					},
				},
				{
					Item: "new-link",
					Ops: []FileOp{
						NewSymlinkOp("some/dest"),
					},
				},
			},
		},
	}

	for desc, test := range tests {
		const target = "target"
		t.Run(desc, func(t *testing.T) {
			plan := New(target, test.specs)

			wantPlan := Plan{Target: target, Tasks: test.wantTasks}

			if diff := cmp.Diff(wantPlan, plan); diff != "" {
				t.Error("Plan: ", diff)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	tests := map[string]struct {
		current *file.State
		planned *file.State
		wantOps []FileOp
	}{
		"no change from nil to nil": {
			wantOps: []FileOp{},
		},
		"no change from link to link, same dest file": {
			current: linkState("../some/dest", 0),
			planned: linkState("../some/dest", 0),
			wantOps: []FileOp{},
		},
		"no change from link to link, same dest dir": {
			current: linkState("../some/dest", fs.ModeDir),
			planned: linkState("../some/dest", fs.ModeDir),
			wantOps: []FileOp{},
		},
		"no change from link to link, same dest link": {
			current: linkState("../some/dest", fs.ModeSymlink),
			planned: linkState("../some/dest", fs.ModeSymlink),
			wantOps: []FileOp{},
		},
		"no change from dir to dir": {
			current: dirState(),
			planned: dirState(),
			wantOps: []FileOp{},
		},
		"from nil to symlink": {
			current: nil,
			planned: linkState("../planned/dest", 0),
			wantOps: []FileOp{
				NewSymlinkOp("../planned/dest"),
			},
		},
		"from nil to dir": {
			current: nil,
			planned: dirState(),
			wantOps: []FileOp{
				MkDirOp,
			},
		},
		"from symlink to dir": {
			current: linkState("some/dest", 0),
			planned: dirState(),
			wantOps: []FileOp{
				RemoveOp,
				MkDirOp,
			},
		},
	}

	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			item := "item"
			spec := Spec{test.current, test.planned}

			gotTask := NewTask(item, spec)

			wantTask := Task{Item: item, Ops: test.wantOps}

			if diff := cmp.Diff(wantTask, gotTask, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("NewTask():\n%s", diff)
			}
		})
	}
}
