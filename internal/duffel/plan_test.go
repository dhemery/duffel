package duffel

import (
	"slices"
	"testing"
)

func TestPlannerTasks(t *testing.T) {
	tests := map[string]struct {
		statuses map[string]Status
		want     []Task
	}{
		"only prior results": {
			statuses: map[string]Status{
				"item1": {Prior: Result{Dest: "item1/prior/dest"}},
				"item2": {Prior: Result{Dest: "item2/prior/dest"}},
				"item3": {Prior: Result{Dest: "item4/prior/dest"}},
			},
			want: []Task{},
		},
		"only planned results": {
			statuses: map[string]Status{
				"item1": {Planned: Result{Dest: "item1/planned/dest"}},
				"item2": {Planned: Result{Dest: "item2/planned/dest"}},
				"item3": {Planned: Result{Dest: "item3/planned/dest"}},
			},
			want: []Task{ // Tasks for all results, sorted by item
				{Item: "item1", Result: Result{Dest: "item1/planned/dest"}},
				{Item: "item2", Result: Result{Dest: "item2/planned/dest"}},
				{Item: "item3", Result: Result{Dest: "item3/planned/dest"}},
			},
		},
		"prior and planned results": {
			statuses: map[string]Status{
				"empty":  {}, // No planned or prior
				"relax":  {Prior: Result{Dest: "prior/dest"}},
				"create": {Planned: Result{Dest: "created/dest"}},
				"change": {
					Prior:   Result{Dest: "prior/dest"},
					Planned: Result{Dest: "changed/dest"},
				},
			},
			want: []Task{ // Tasks only for planned results, sorted by item
				{Item: "change", Result: Result{Dest: "changed/dest"}},
				{Item: "create", Result: Result{Dest: "created/dest"}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			planner := Planner(test.statuses)

			got := planner.Tasks()

			if !slices.Equal(got, test.want) {
				t.Errorf("want tasks %v, got %v", test.want, got)
			}
		})
	}
}

func TestPlannerCreate(t *testing.T) {
	item := "item"
	dest := "task/dest"

	planner := Planner{}

	got := planner.Status(item)
	want := Status{}
	if got != want {
		t.Fatalf("before create want status %v, got %v", want, got)
	}

	result := Result{Dest: dest}
	planner.Create(item, result)

	want = Status{Planned: result}
	got = planner.Status(item)
	if got != want {
		t.Fatalf("after create want status %v, got %v", want, got)
	}
}

func TestStatusWillExist(t *testing.T) {
	var (
		priorExist   = Result{Dest: "prior/exist/dest"}
		plannedExist = Result{Dest: "planned/exist/dest"}
	)
	tests := []struct {
		status Status
		want   bool
	}{
		{status: Status{}, want: false},
		{status: Status{Prior: priorExist}, want: true},
		{status: Status{Planned: plannedExist}, want: true},
		{status: Status{Prior: priorExist, Planned: plannedExist}, want: true},
	}

	for _, test := range tests {
		status := test.status
		got := status.WillExist()
		if got != test.want {
			t.Errorf("%v want %t, got %t", status, test.want, got)
		}
	}
}

func TestResultExists(t *testing.T) {
	tests := []struct {
		result Result
		want   bool
	}{
		{result: Result{Dest: ""}, want: false},
		{result: Result{Dest: "not/empty"}, want: true},
	}

	for _, test := range tests {
		status := test.result
		got := status.Exists()
		if got != test.want {
			t.Errorf("%v want %t, got %t", status, test.want, got)
		}
	}
}
