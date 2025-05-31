package duffel

import (
	"path"
	"testing"
)

func TestCreateLink(t *testing.T) {
	pkg := "pkg"
	item := "item"
	linkPrefix := "../.."

	planner := &Planner{LinkPrefix: linkPrefix}
	planner.CreateLink(pkg, item)

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 1 {
		t.Fatalf("want 1 planned task, got %d: %#v", len(gotTasks), gotTasks)
	}

	gotTask := gotTasks[0]
	wantTask := CreateLink{Action: "link", Path: item, Dest: path.Join(linkPrefix, pkg, item)}
	if gotTask != wantTask {
		t.Fatalf("want task %#v, got %#v", wantTask, gotTask)
	}
}
