package duffel

import (
	"testing"
)

func TestCreate(t *testing.T) {
	item := "item"
	dest := "task/dest"

	planner := NewPlanner("", "")

	got := planner.Status(item)
	if got != nil {
		t.Fatalf("before create want status %v, got %v", nil, got)
	}

	result := &Result{Dest: dest}
	planner.Create(item, result)

	got = planner.Status(item)
	want := &Status{Planned: result}
	if got == nil {
		t.Fatalf("after create want status %v, got %v", *want, nil)
	}
	if *got != *want {
		t.Fatalf("after create want status %v, got %v", *want, *got)
	}
}
