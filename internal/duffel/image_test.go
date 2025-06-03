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
		"only existing states": {
			statuses: map[string]Status{
				"item1": {Existing: State{Dest: "item1/existing/dest"}},
				"item2": {Existing: State{Dest: "item2/existing/dest"}},
				"item3": {Existing: State{Dest: "item4/existing/dest"}},
			},
			want: []Task{},
		},
		"only planned states": {
			statuses: map[string]Status{
				"item1": {Planned: State{Dest: "item1/planned/dest"}},
				"item2": {Planned: State{Dest: "item2/planned/dest"}},
				"item3": {Planned: State{Dest: "item3/planned/dest"}},
			},
			want: []Task{ // Tasks for all states, sorted by item
				{Item: "item1", State: State{Dest: "item1/planned/dest"}},
				{Item: "item2", State: State{Dest: "item2/planned/dest"}},
				{Item: "item3", State: State{Dest: "item3/planned/dest"}},
			},
		},
		"existing and planned states": {
			statuses: map[string]Status{
				"empty":  {}, // No planned or existing
				"relax":  {Existing: State{Dest: "existing/dest"}},
				"create": {Planned: State{Dest: "created/dest"}},
				"change": {
					Existing: State{Dest: "existing/dest"},
					Planned:  State{Dest: "changed/dest"},
				},
			},
			want: []Task{ // Tasks only for planned states, sorted by item
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

	got := image.Status(item)
	want := Status{}
	if got != want {
		t.Fatalf("before create want status %v, got %v", want, got)
	}

	state := State{Dest: dest}
	image.Create(item, state)

	want = Status{Planned: state}
	got = image.Status(item)
	if got != want {
		t.Fatalf("after create want status %v, got %v", want, got)
	}
}

func TestStatusWillExist(t *testing.T) {
	var (
		existingState = State{Dest: "existing/dest"}
		plannedState  = State{Dest: "planned/dest"}
	)
	tests := []struct {
		status Status
		want   bool
	}{
		{status: Status{}, want: false},
		{status: Status{Existing: existingState}, want: true},
		{status: Status{Planned: plannedState}, want: true},
		{status: Status{Existing: existingState, Planned: plannedState}, want: true},
	}

	for _, test := range tests {
		status := test.status
		got := status.WillExist()
		if got != test.want {
			t.Errorf("%v want %t, got %t", status, test.want, got)
		}
	}
}

func TestStateExists(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{state: State{Dest: ""}, want: false},
		{state: State{Dest: "not/empty"}, want: true},
	}

	for _, test := range tests {
		status := test.state
		got := status.Exists()
		if got != test.want {
			t.Errorf("%v want %t, got %t", status, test.want, got)
		}
	}
}
