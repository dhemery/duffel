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
	const pkgName = "pkg"

	planner := &Planner{}
	mode := fs.ModeDir | 0o755 // Dir
	entry := dirEntry{pkgName, mode, nil}
	pkgDir := path.Join("path/to/source", pkgName)

	visit := PlanInstallPackage(planner, pkgDir, pkgName)

	err := visit(pkgDir, entry, nil)
	if err != nil {
		t.Error(err)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 0 {
		t.Fatalf("want 0 tasks, got %d: %#v", len(gotTasks), gotTasks)
	}
}

func TestInstallVisitPkgDirWithError(t *testing.T) {
	const pkgName = "pkg"

	planner := &Planner{}
	pkgDir := path.Join("path/to/source", pkgName)
	givenErr := errors.New("custom error")

	visit := PlanInstallPackage(planner, pkgDir, pkgName)

	gotErr := visit(pkgDir, nil, givenErr)
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
		linkPrefix = "../source" // Prepended by the planner onto each link dest
		pkgName    = "pkg"
		itemName   = "item"
	)

	planner := &Planner{LinkPrefix: linkPrefix}

	mode := fs.FileMode(0o644) // Regular file
	entry := dirEntry{itemName, mode, nil}
	pkgDir := path.Join("path/to/source", pkgName)
	itemPath := path.Join(pkgDir, itemName)

	visit := PlanInstallPackage(planner, pkgDir, pkgName)

	err := visit(itemPath, entry, nil)
	if err != nil {
		t.Fatal(err)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 1 {
		t.Fatalf("want 1 task, got %d: %#v", len(gotTasks), gotTasks)
	}

	wantTask := CreateLink{
		Action: "link",
		Path:   itemName,
		Dest:   path.Join(linkPrefix, pkgName, itemName),
	}
	gotTask := gotTasks[0]
	if gotTask != wantTask {
		t.Errorf("want task %#v, got %#v", wantTask, gotTask)
	}
}

func TestInstallVisitItemWithError(t *testing.T) {
	const (
		pkgName  = "pkg"
		itemName = "item"
	)

	planner := &Planner{}

	pkgDir := path.Join("path/to/source", pkgName)
	itemPath := path.Join(pkgDir, itemName)
	givenErr := errors.New("custom error")

	visit := PlanInstallPackage(planner, pkgDir, pkgName)

	gotErr := visit(itemPath, nil, givenErr)
	if !errors.Is(gotErr, givenErr) {
		t.Errorf("want error %q, got %q", givenErr, gotErr)
	}

	gotTasks := planner.Plan.Tasks
	if len(gotTasks) != 0 {
		t.Fatalf("want 0 tasks, got %d: %#v", len(gotTasks), gotTasks)
	}
}
