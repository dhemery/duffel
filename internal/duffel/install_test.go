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

func TestVisitInstall(t *testing.T) {
	const (
		source         = "path/to/source"   // Parent dir of package being walked
		pkg            = "pkg"              // Package dir being walked, relative to source
		targetToSource = "target/to/source" // Given to walk func to use in link dests
	)
	visitErr := errors.New("error passed to visit")

	tests := map[string]struct {
		item       string // Item being visited, relative to pkg dir
		givenErr   error  // Error passed to visit
		status     Status // Planner status before visit
		wantStatus Status // Planner status after visit
		wantErr    error  // Returned by visit
	}{
		"pkg dir": {
			item:       ".",
			givenErr:   nil,
			wantErr:    nil,
			wantStatus: Status{}, // Plans no action
		},
		"pkg dir and given error": {
			item:       ".",
			givenErr:   visitErr,
			wantErr:    visitErr,
			wantStatus: Status{},
		},
		"item with no status": {
			item:       "item",
			givenErr:   nil,
			status:     Status{},
			wantErr:    nil,
			wantStatus: Status{Planned: Result{Dest: path.Join(targetToSource, pkg, "item")}},
		},
		"item and given error": {
			item:       "item",
			givenErr:   visitErr,
			wantErr:    visitErr,
			wantStatus: Status{},
		},
		"preexisting item": {
			item:       "item",
			status:     Status{Prior: Result{Dest: "prior/link/dest"}},
			wantErr:    &ErrConflict{},
			wantStatus: Status{Prior: Result{Dest: "prior/link/dest"}}, // Unchanged
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sourcePkg := path.Join(source, pkg)
			visitPath := path.Join(sourcePkg, test.item)

			planner := Planner{}
			if test.status.WillExist() {
				planner[test.item] = test.status
			}

			visit := PlanInstallPackage(planner, targetToSource, sourcePkg, pkg)

			gotErr := visit(visitPath, nil, test.givenErr)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("want error %#v, got %#v", test.wantErr, gotErr)
			}

			gotStatus := planner.Status(test.item)
			if gotStatus != test.wantStatus {
				t.Errorf("want status %v, got %v", test.wantStatus, gotStatus)
			}
		})
	}
}
