package plan

import (
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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
