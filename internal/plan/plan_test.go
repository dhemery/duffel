package plan

import (
	"iter"
	"maps"
	"slices"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/item"
)

type specs interface {
	All() iter.Seq2[string, item.Spec]
}

type specMap map[string]item.Spec

func (m specMap) ByItem() iter.Seq2[string, item.Spec] {
	return func(yield func(string, item.Spec) bool) {
		for _, key := range slices.Sorted(maps.Keys(m)) {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}

func TestNewPlan(t *testing.T) {
	tests := map[string]struct {
		index     specMap
		wantTasks []Task
	}{
		"only current states": {
			index: map[string]item.Spec{
				"item1": {Current: &file.State{Dest: "item1/current/dest"}},
				"item2": {Current: &file.State{Dest: "item2/current/dest"}},
				"item3": {Current: &file.State{Dest: "item4/current/dest"}},
			},
			wantTasks: []Task{},
		},
		"only desired states": {
			index: map[string]item.Spec{
				"item1": {Desired: &file.State{Dest: "item1/desired/dest"}},
				"item2": {Desired: &file.State{Dest: "item2/desired/dest"}},
				"item3": {Desired: &file.State{Dest: "item3/desired/dest"}},
			},
			wantTasks: []Task{ // Tasks for all states, sorted by item
				{Item: "item1", State: file.State{Dest: "item1/desired/dest"}},
				{Item: "item2", State: file.State{Dest: "item2/desired/dest"}},
				{Item: "item3", State: file.State{Dest: "item3/desired/dest"}},
			},
		},
		"current and desired states": {
			index: map[string]item.Spec{
				"empty":  {}, // No current or desired state
				"relax":  {Current: &file.State{Dest: "current/dest"}},
				"create": {Desired: &file.State{Dest: "created/dest"}},
				"change": {
					Current: &file.State{Dest: "current/dest"},
					Desired: &file.State{Dest: "changed/dest"},
				},
			},
			wantTasks: []Task{ // Tasks only for desired states, sorted by item
				{Item: "change", State: file.State{Dest: "changed/dest"}},
				{Item: "create", State: file.State{Dest: "created/dest"}},
			},
		},
	}

	for name, test := range tests {
		const target = "path/to/target"

		t.Run(name, func(t *testing.T) {
			got := New(target, test.index)

			if got.Target != target {
				t.Errorf("want target %q, got %q", target, got.Target)
			}
			if !slices.Equal(got.Tasks, test.wantTasks) {
				t.Errorf("want tasks %v, got %v", test.wantTasks, got)
			}
		})
	}
}
