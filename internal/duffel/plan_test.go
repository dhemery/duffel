package duffel

import (
	"slices"
	"testing"
)

func TestNewPlan(t *testing.T) {
	tests := map[string]struct {
		targetGap TargetGap
		wantTasks []Task
	}{
		"only current states": {
			targetGap: TargetGap{
				"item1": {Current: &FileState{Dest: "item1/current/dest"}},
				"item2": {Current: &FileState{Dest: "item2/current/dest"}},
				"item3": {Current: &FileState{Dest: "item4/current/dest"}},
			},
			wantTasks: []Task{},
		},
		"only desired states": {
			targetGap: TargetGap{
				"item1": {Desired: &FileState{Dest: "item1/desired/dest"}},
				"item2": {Desired: &FileState{Dest: "item2/desired/dest"}},
				"item3": {Desired: &FileState{Dest: "item3/desired/dest"}},
			},
			wantTasks: []Task{ // Tasks for all states, sorted by item
				{Item: "item1", FileState: FileState{Dest: "item1/desired/dest"}},
				{Item: "item2", FileState: FileState{Dest: "item2/desired/dest"}},
				{Item: "item3", FileState: FileState{Dest: "item3/desired/dest"}},
			},
		},
		"current and desired states": {
			targetGap: TargetGap{
				"empty":  {}, // No current or desired state
				"relax":  {Current: &FileState{Dest: "current/dest"}},
				"create": {Desired: &FileState{Dest: "created/dest"}},
				"change": {
					Current: &FileState{Dest: "current/dest"},
					Desired: &FileState{Dest: "changed/dest"},
				},
			},
			wantTasks: []Task{ // Tasks only for desired states, sorted by item
				{Item: "change", FileState: FileState{Dest: "changed/dest"}},
				{Item: "create", FileState: FileState{Dest: "created/dest"}},
			},
		},
	}

	for name, test := range tests {
		const target = "path/to/target"

		t.Run(name, func(t *testing.T) {
			got := NewPlan(target, test.targetGap)

			if got.Target != target {
				t.Errorf("want target %q, got %q", target, got.Target)
			}
			if !slices.Equal(got.Tasks, test.wantTasks) {
				t.Errorf("want tasks %v, got %v", test.wantTasks, got)
			}
		})
	}
}
