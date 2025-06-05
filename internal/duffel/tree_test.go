package duffel

import (
	"io/fs"
	"reflect"
	"slices"
	"testing"
)

func TestTargetTreeTasks(t *testing.T) {
	tests := map[string]struct {
		statuses map[string]Status
		want     []Task
	}{
		"only current states": {
			statuses: map[string]Status{
				"item1": {Current: &State{Dest: "item1/current/dest"}},
				"item2": {Current: &State{Dest: "item2/current/dest"}},
				"item3": {Current: &State{Dest: "item4/current/dest"}},
			},
			want: []Task{},
		},
		"only desired states": {
			statuses: map[string]Status{
				"item1": {Desired: &State{Dest: "item1/desired/dest"}},
				"item2": {Desired: &State{Dest: "item2/desired/dest"}},
				"item3": {Desired: &State{Dest: "item3/desired/dest"}},
			},
			want: []Task{ // Tasks for all states, sorted by item
				{Item: "item1", State: State{Dest: "item1/desired/dest"}},
				{Item: "item2", State: State{Dest: "item2/desired/dest"}},
				{Item: "item3", State: State{Dest: "item3/desired/dest"}},
			},
		},
		"current and desired states": {
			statuses: map[string]Status{
				"empty":  {}, // No current or desired state
				"relax":  {Current: &State{Dest: "current/dest"}},
				"create": {Desired: &State{Dest: "created/dest"}},
				"change": {
					Current: &State{Dest: "current/dest"},
					Desired: &State{Dest: "changed/dest"},
				},
			},
			want: []Task{ // Tasks only for desired states, sorted by item
				{Item: "change", State: State{Dest: "changed/dest"}},
				{Item: "create", State: State{Dest: "created/dest"}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tree := TargetTree(test.statuses)

			got := tree.Tasks()

			if !slices.Equal(got, test.want) {
				t.Errorf("want tasks %v, got %v", test.want, got)
			}
		})
	}
}

func TestTargetSet(t *testing.T) {
	item := "item"

	tree := TargetTree{}

	got, ok := tree.Status(item)
	if ok {
		t.Fatalf("before create, want !ok, got ok status %v", got)
	}

	status := Status{
		Current: &State{Mode: fs.ModeSymlink, Dest: "current/dest"},
		Desired: &State{Mode: fs.ModeDir | 0o755},
	}
	tree.Set(item, status)

	got, ok = tree.Status(item)
	if got != status || !ok {
		t.Fatalf("after set want ok status %v\ngot ok %t, status %v", status, ok, got)
	}
}

func TestNewStatus(t *testing.T) {
	mode := fs.ModeDir | 0o755
	dest := "my/dest"

	got := NewStatus(mode, dest)

	// Records the given mode and dest as both the current and desired states
	want := Status{
		Current: &State{Mode: mode, Dest: dest},
		Desired: &State{Mode: mode, Dest: dest},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("new status:\nwant %s\ngot  %s", want, got)
	}
}
