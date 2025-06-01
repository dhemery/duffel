package duffel

import (
	"errors"
	"io/fs"
	"path"
	"testing"
)

type dirEntry struct {
	name string
	mode fs.FileMode
	info fs.FileInfo
}

func (d dirEntry) IsDir() bool {
	return d.mode.IsDir()
}

func (d dirEntry) Info() (fs.FileInfo, error) {
	if d.info == nil {
		return nil, fs.ErrNotExist
	}
	return d.info, nil
}

func (d dirEntry) Name() string {
	return d.name
}

func (d dirEntry) Type() fs.FileMode {
	return d.mode & fs.ModeType
}

func TestInstallVisitPkgDir(t *testing.T) {
	const (
		pkg    = "pkg"
		source = "path/to/source"
	)

	planner := NewPlanner("", "")
	mode := fs.ModeDir | 0o755 // Dir
	entry := dirEntry{pkg, mode, nil}
	sourcePkg := path.Join(source, pkg)

	visit := PlanInstallPackage(planner, sourcePkg, pkg)

	err := visit(sourcePkg, entry, nil)
	if err != nil {
		t.Error(err)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 0 {
		t.Fatalf("want 0 tasks, got %d: %#v", len(gotTasks), gotTasks)
	}
}

func TestInstallVisitPkgDirWithError(t *testing.T) {
	const (
		pkg    = "pkg"
		source = "path/to/source"
	)

	planner := NewPlanner("", "")
	sourcePkg := path.Join(source, pkg)
	givenErr := errors.New("custom error")

	visit := PlanInstallPackage(planner, sourcePkg, pkg)

	gotErr := visit(sourcePkg, nil, givenErr)
	if !errors.Is(gotErr, givenErr) {
		t.Errorf("want error %q, got %q", givenErr, gotErr)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 0 {
		t.Fatalf("want 0 tasks, got %d: %#v", len(gotTasks), gotTasks)
	}
}

func TestInstallVisitItem(t *testing.T) {
	const (
		source         = "path/to/source"
		pkg            = "pkg"
		item           = "item"
		targetToSource = "../source" // Prepended by the planner onto each link dest
	)

	planner := NewPlanner("", targetToSource)

	mode := fs.FileMode(0o644) // Regular file
	entry := dirEntry{item, mode, nil}
	sourcePkg := path.Join(source, pkg)
	sourcePkgItem := path.Join(sourcePkg, item)

	visit := PlanInstallPackage(planner, sourcePkg, pkg)

	err := visit(sourcePkgItem, entry, nil)
	if err != nil {
		t.Fatal(err)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 1 {
		t.Fatalf("want 1 task, got %d: %#v", len(gotTasks), gotTasks)
	}

	wantTask := CreateLink{
		Action: "link",
		Item:   item,
		Dest:   path.Join(targetToSource, pkg, item),
	}
	gotTask := gotTasks[0]
	if gotTask != wantTask {
		t.Errorf("want task %#v, got %#v", wantTask, gotTask)
	}
}

func TestInstallVisitItemWithError(t *testing.T) {
	const (
		source         = "path/to/source"
		pkg            = "pkg"
		item           = "item"
		targetToSource = "../source"
	)

	planner := NewPlanner("", targetToSource)

	sourcePkg := path.Join(source, pkg)
	sourcePkgItem := path.Join(sourcePkg, item)
	givenErr := errors.New("custom error")

	visit := PlanInstallPackage(planner, sourcePkg, pkg)

	gotErr := visit(sourcePkgItem, nil, givenErr)
	if !errors.Is(gotErr, givenErr) {
		t.Errorf("want error %q, got %q", givenErr, gotErr)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 0 {
		t.Fatalf("want 0 tasks, got %d: %#v", len(gotTasks), gotTasks)
	}
}
