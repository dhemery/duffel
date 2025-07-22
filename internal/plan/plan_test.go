package plan

import (
	"io/fs"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewTask(t *testing.T) {
	tests := map[string]struct {
		item string
		spec Spec
		want Task
	}{
		"no change from nil to nil": {
			item: "item",
			spec: Spec{
				Current: nil,
				Planned: nil,
			},
			want: Task{Item: "item", Ops: []FileOp{}},
		},
		"no change from link to link, same dest file": {
			item: "item",
			spec: Spec{
				Current: linkState("../some/dest", 0),
				Planned: linkState("../some/dest", 0),
			},
			want: Task{Item: "item", Ops: []FileOp{}},
		},
		"no change from link to link, same dest dir": {
			item: "item",
			spec: Spec{
				Current: linkState("../some/dest", fs.ModeDir),
				Planned: linkState("../some/dest", fs.ModeDir),
			},
			want: Task{Item: "item", Ops: []FileOp{}},
		},
		"no change from link to link, same dest link": {
			item: "item",
			spec: Spec{
				Current: linkState("../some/dest", fs.ModeSymlink),
				Planned: linkState("../some/dest", fs.ModeSymlink),
			},
			want: Task{Item: "item", Ops: []FileOp{}},
		},
		"no change from dir to dir": {
			item: "item",
			spec: Spec{
				Current: dirState(),
				Planned: dirState(),
			},
			want: Task{Item: "item", Ops: []FileOp{}},
		},
	}
	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			got := NewTask(test.item, test.spec)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("NewTask():\n%s", diff)
			}
		})
	}
}
