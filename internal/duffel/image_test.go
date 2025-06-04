package duffel

import (
	"slices"
	"testing"
)

func TestImageTasks(t *testing.T) {
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
			image := Image(test.statuses)

			got := image.Tasks()

			if !slices.Equal(got, test.want) {
				t.Errorf("want tasks %v, got %v", test.want, got)
			}
		})
	}
}

func TestImageCreate(t *testing.T) {
	item := "item"
	dest := "task/dest"

	image := Image{}

	got, ok := image.Status(item)
	if ok {
		t.Fatalf("before create, want !ok, got ok status %v", got)
	}

	state := &State{Dest: dest}
	image.Create(item, state)

	want := Status{Desired: state}
	got, ok = image.Status(item)
	if got != want || !ok {
		t.Fatalf("after create want ok status %v\ngot ok %t, status %v", want, ok, got)
	}
}
