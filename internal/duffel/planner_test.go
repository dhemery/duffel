package duffel

import (
	"path"
	"testing"
)

func TestCreateLink(t *testing.T) {
	pkg := "pkg"
	item := "item"
	targetToSource := "../.."

	planner := NewPlanner("", targetToSource)

	planner.CreateLink(pkg, item)

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 1 {
		t.Fatalf("want 1 planned task, got %d: %#v", len(gotTasks), gotTasks)
	}

	gotTask := gotTasks[0]
	wantTask := CreateLink{Action: "link", Item: item, Dest: path.Join(targetToSource, pkg, item)}
	if gotTask != wantTask {
		t.Fatalf("want task %#v, got %#v", wantTask, gotTask)
	}
}

func TestTargetItemExists(t *testing.T) {
	const (
		pkg  = "pkg"
		item = "item"
	)
	planner := NewPlanner("", "")

	exists := planner.Exists(item)
	if exists {
		t.Errorf("before create link, %q exists want %t, got %t", item, !exists, exists)
	}

	planner.CreateLink(pkg, item)

	exists = planner.Exists(item)
	if !exists {
		t.Errorf("after create link, %q exists want %t, got %t", item, !exists, exists)
	}
}
